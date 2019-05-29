# OD Alerting CLI

Alerting CLI enables to manage and organize Elasticsearch Alerting monitors through YAML file.

## Why would I want to manage my monitors in YAML files?

Excellent question. So many reasons:

* You can store your monitors in source control so you can track changes (who changed this monitor?), and perform code reviews.
* Enable through automated pipelines.
* It is very easy to accidentally make changes to monitors on UI or through API. You can always refer back on this and update existing monitors.
* You may have multiple stages / regions and would like have monitors running on each stage / region.

## Installation

Download the distributions from [here]()

## Getting Started

Currently, this CLI doesn't support how the destinations are managed. This will be supported in up-coming versions.
For now, after installing you can run the commands to sync your destinations.

```
alerting-config-cli sync --destinations
```

This command will create auto-generated destinations file with names and destinationId , so that they're easy to refer inside monitors.

### Start with existing remote monitors

```
alerting-config-cli sync --monitors
```

This command will create `monitors.yaml` and write them to local files and you can start off managing your monitors.

### Monitors diff

```
alerting-config-cli diff

```

This command will compare your local monitors with remote monitors and dispaly delta.

### publish monitors to remote ES

Publish local monitors to remote elasticsearch cluster:

- will run, validate updated / new monitors
- will create new monitors
- will update existing monitors (so it could override what you were doing if
  you edit an existing monitor in Kibana alerting or by any other way)
- This cli will not delete anything from remote you will need to add `--delete` flag to delete untracked monitors


Sample monitor

```
- name: 'Sample Alerting monitor'
  type: 'monitor'
  schedule:
    period:
      interval: 10
      unit: MINUTES
  enabled: true
  inputs:
    - search:
        indices: ["log*"]
        query: # This is valid Elasticsearch query
          size: 0
          query:
            match_all: {
              boost: 1.0
            }
  triggers:
    - name: '500'
      severity: '2'
      condition: "return true"
      actions:
        - name: Sample Action
          destinationId: test_my_destination #This destination should be avialble in destinations.yaml file otherwise it will not work.
          subject:'Error'
          message: 'There is an error'
```

##Note

* You might see the diff in queries after pushing new monitors, this could be because Alerting uses ESQuery builder,which adds some defaults to improve the query. Currently only solution is to adhere to remote query and update yaml file manually. You could currently do this with either `diff` and applying diff.

## ToDO

* Add support of providing validation against custom TLS certificate.
* Support AWS Elasticsearch
*