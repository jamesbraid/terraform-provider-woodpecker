package internal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/Kichiyaki/terraform-provider-woodpecker/internal/woodpecker"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type repositoryClientStub struct {
	woodpecker.Client
	repo  woodpecker.Repo
	calls []string
	err   error
}

func (c *repositoryClientStub) RepoLookup(fullName string) (*woodpecker.Repo, error) {
	c.calls = append(c.calls, "lookup")
	if fullName != c.repo.FullName || !c.repo.IsActive {
		return nil, &woodpecker.ClientError{StatusCode: http.StatusNotFound}
	}
	return &c.repo, nil
}

func (c *repositoryClientStub) RepoList(opt woodpecker.RepoListOptions) ([]*woodpecker.Repo, error) {
	if !opt.All {
		return nil, errors.New("repository list must include inactive repositories")
	}
	c.calls = append(c.calls, "list")
	if c.err != nil {
		return nil, c.err
	}
	return []*woodpecker.Repo{&c.repo}, nil
}

func (c *repositoryClientStub) RepoPost(opt woodpecker.RepoPostOptions) (*woodpecker.Repo, error) {
	if opt.ForgeRemoteID != c.repo.ForgeRemoteID {
		return nil, errors.New("wrong forge remote ID")
	}
	c.calls = append(c.calls, "activate")
	c.repo.IsActive = true
	return &c.repo, nil
}

func (c *repositoryClientStub) RepoPatch(id int64, patch *woodpecker.RepoPatch) (*woodpecker.Repo, error) {
	if id != c.repo.ID || patch.RequireApproval == nil {
		return nil, errors.New("wrong repository ID or missing approval mode")
	}
	c.calls = append(c.calls, "patch")
	c.repo.RequireApproval = *patch.RequireApproval
	return &c.repo, nil
}

func (c *repositoryClientStub) RepoDel(id int64) error {
	if id != c.repo.ID {
		return errors.New("wrong repository ID")
	}
	c.calls = append(c.calls, "deactivate")
	c.repo.IsActive = false
	return nil
}

func TestRepositoryApprovalPatch(t *testing.T) {
	model := repositoryModel{
		RequireApproval:              types.StringValue(woodpecker.ApprovalModeAllEvents.String()),
		CancelPreviousPipelineEvents: types.SetNull(types.StringType),
		NetrcTrustedPlugins:          types.SetNull(types.StringType),
		ApprovalAllowedUsers:         types.SetNull(types.StringType),
		Trusted:                      types.ObjectNull(repositoryModelTrustedAttributes),
	}
	patch, diags := model.toWoodpeckerPatch(context.Background())
	if diags.HasError() {
		t.Fatal(diags)
	}
	encoded, err := json.Marshal(patch)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["require_approval"] != woodpecker.ApprovalModeAllEvents.String() {
		t.Fatalf("approval mode missing from patch: %s", encoded)
	}

	model.RequireApproval = types.StringNull()
	patch, diags = model.toWoodpeckerPatch(context.Background())
	if diags.HasError() {
		t.Fatal(diags)
	}
	if patch.RequireApproval != nil {
		t.Fatal("unset approval mode must not be patched")
	}
}

func TestRepositoryApprovalNoneAccepted(t *testing.T) {
	ctx := context.Background()
	var schemaResponse resource.SchemaResponse
	(&repositoryResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	attribute := schemaResponse.Schema.Attributes["require_approval"].(schema.StringAttribute)
	for _, value := range []struct {
		mode      string
		wantError bool
	}{
		{mode: woodpecker.ApprovalModeNone.String()},
		{mode: "invalid", wantError: true},
	} {
		var response validator.StringResponse
		attribute.Validators[0].ValidateString(ctx, validator.StringRequest{
			Path: path.Root("require_approval"), ConfigValue: types.StringValue(value.mode),
		}, &response)
		if response.Diagnostics.HasError() != value.wantError {
			t.Fatalf("validation for %q: %v", value.mode, response.Diagnostics)
		}
	}

	model := repositoryModel{
		RequireApproval:              types.StringValue(woodpecker.ApprovalModeNone.String()),
		CancelPreviousPipelineEvents: types.SetNull(types.StringType),
		NetrcTrustedPlugins:          types.SetNull(types.StringType),
		ApprovalAllowedUsers:         types.SetNull(types.StringType),
		Trusted:                      types.ObjectNull(repositoryModelTrustedAttributes),
	}
	patch, diags := model.toWoodpeckerPatch(ctx)
	if diags.HasError() || patch.RequireApproval == nil || *patch.RequireApproval != woodpecker.ApprovalModeNone {
		t.Fatalf("none must be sent in the repository patch: patch=%+v diagnostics=%v", patch, diags)
	}
}

func TestRepositoryActiveTransitions(t *testing.T) {
	approval := woodpecker.ApprovalModeAllEvents
	patch := &woodpecker.RepoPatch{RequireApproval: &approval}
	for _, tc := range []struct {
		name  string
		old   bool
		want  bool
		calls []string
	}{
		{name: "deactivate", old: true, want: false, calls: []string{"patch", "deactivate", "lookup", "list"}},
		{name: "reactivate", old: false, want: true, calls: []string{"activate", "patch"}},
		{name: "retain active", old: true, want: true, calls: []string{"patch"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &repositoryClientStub{repo: woodpecker.Repo{
				ID: 7, ForgeRemoteID: "23", FullName: "infra/example", IsActive: tc.old,
			}}
			r := &repositoryResource{client: client}
			previous := &repositoryModel{
				ID: types.Int64Value(7), ForgeRemoteID: types.StringValue("23"),
				FullName: types.StringValue("infra/example"), IsActive: types.BoolValue(tc.old),
			}
			planned := &repositoryModel{IsActive: types.BoolValue(tc.want)}
			repo, err := r.updateRepository(previous, planned, patch)
			if err != nil {
				t.Fatal(err)
			}
			if repo.IsActive != tc.want || repo.RequireApproval != approval {
				t.Fatalf("unexpected repository after update: active=%t approval=%s", repo.IsActive, repo.RequireApproval)
			}
			if !reflect.DeepEqual(client.calls, tc.calls) {
				t.Fatalf("calls = %v, want %v", client.calls, tc.calls)
			}
		})
	}
}

func TestRepositoryByNameStopsOnListError(t *testing.T) {
	client := &repositoryClientStub{err: errors.New("repository listing failed")}
	r := &repositoryResource{client: client}
	if _, err := r.repositoryByName("infra/example"); err == nil {
		t.Fatal("repository lookup must return the list error")
	}
}
