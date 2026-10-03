# Data dictionary

Synthetic catalog used by `finguard seed`. Labels below are ground truth for tests. They are not included in model prompts.

| Scenario | Account | Expected rules | Case |
| --- | --- | --- | --- |
| normal_activity | acct_ava | none | no |
| velocity_burst | acct_milo | velocity | yes |
| amount_deviation | acct_lee | amount_deviation | yes |
| repeated_failures | acct_noor | repeated_failures | yes |
| shared_device | acct_kim | shared_device | yes |
| geographic_inconsistency | acct_quinn | geographic_inconsistency | yes |
| merchant_novelty | acct_wynn | merchant_novelty | yes |
| takeover_indicators | acct_ren | amount, geo, merchant novelty, repeated failures | yes |
| legitimate_travel | acct_sam | none | no |
| demo_mixed_devices | acct_demo | velocity, two devices | yes |

Amounts are minor units of USD in the seed (100 minor units = 1.00). People and merchants are fictional. Memos are short synthetic notes.

`acct_park` shares `dev_shared` with `acct_kim` and is the earlier observation, so the shared-device signal is attached when the second account appears.
