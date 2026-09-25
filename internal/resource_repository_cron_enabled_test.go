package internal_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestRepositoryCronEnabled(t *testing.T) {
	version, err := woodpeckerClient.Version()
	if err != nil {
		t.Fatal(err)
	}
	if semver.MustParse(version.Version).LessThan(semver.MustParse("3.13.0")) {
		t.Skip("cron enabled requires Woodpecker 3.13.0 or later")
	}

	t.Parallel()
	repo := activateRepo(t, createRepo(t))
	name := uuid.NewString()

	config := func(schedule, enabled string) string {
		return fmt.Sprintf(`
resource "woodpecker_repository_cron" "test_cron" {
	repository_id = %d
	name = "%s"
	schedule = "%s"
	%s
}

data "woodpecker_repository_cron" "test_cron" {
	repository_id = woodpecker_repository_cron.test_cron.repository_id
	id = woodpecker_repository_cron.test_cron.id
}
`, repo.ID, name, schedule, enabled)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: checkRepositoryCronResourceDestroy(map[int64][]string{
			repo.ID: {name},
		}),
		Steps: []resource.TestStep{
			{
				Config: config("@daily", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectUnknownValue("woodpecker_repository_cron.test_cron", tfjsonpath.New("enabled")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("woodpecker_repository_cron.test_cron", "enabled", "true"),
					resource.TestCheckResourceAttr("data.woodpecker_repository_cron.test_cron", "enabled", "true"),
				),
			},
			{
				ResourceName:        "woodpecker_repository_cron.test_cron",
				ImportState:         true,
				ImportStateIdPrefix: strconv.FormatInt(repo.ID, 10) + "/",
				ImportStateVerify:   true,
			},
			{
				Config: config("@daily", "enabled = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("woodpecker_repository_cron.test_cron", "enabled", "false"),
					resource.TestCheckResourceAttr("data.woodpecker_repository_cron.test_cron", "enabled", "false"),
				),
			},
			{
				ResourceName:        "woodpecker_repository_cron.test_cron",
				ImportState:         true,
				ImportStateIdPrefix: strconv.FormatInt(repo.ID, 10) + "/",
				ImportStateVerify:   true,
			},
			{
				Config: config("@every 5m", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("woodpecker_repository_cron.test_cron", "enabled", "false"),
					resource.TestCheckResourceAttr("data.woodpecker_repository_cron.test_cron", "enabled", "false"),
				),
			},
			{
				Config: config("@daily", "enabled = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("woodpecker_repository_cron.test_cron", "enabled", "true"),
					resource.TestCheckResourceAttr("data.woodpecker_repository_cron.test_cron", "enabled", "true"),
				),
			},
		},
	})
}
