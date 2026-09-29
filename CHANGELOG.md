# Changelog

## 0.6.1

- Repository cron jobs expose an `enabled` setting. New jobs default to
  enabled on Woodpecker 3.13 or later, and imported jobs retain their live
  setting.
- Repository `require_approval` now updates, including when set to `none`.
  Existing repositories can be activated or deactivated with `is_active`.
  Import an inactive repository before managing it with `is_active = false`.
  Creating an inactive repository is not supported.
