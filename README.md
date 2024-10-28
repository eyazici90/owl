# owl

owl is cli tool for prometheus & grafana. 
It analyses your prometheus metrics, rules(recording & alerting) & dashboards to give you some insights.

 
`owl --help`
```commandline
USAGE:
owl [global options] command [command options]

VERSION:
v0.0.1

DESCRIPTION:
Observability CLI

COMMANDS:
rules       
metrics     
dashboards  
help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
--log-level value  (default: "info")
--help, -h         show help
--version, -v      print the version

```

`owl rules --help`

```commandline
NAME:
   owl rules

USAGE:
   owl rules command [command options]

COMMANDS:
   export   Exports prom rules to csv file
   idle     Scans prom rules to find ones that are missing metrics
   slowest  Scans prom rules to find slowest ones based on evaluation durations
   help, h  Shows a list of commands or help for one command


```

`owl metrics --help`
```commandline
NAME:
   owl metrics

USAGE:
   owl metrics command [command options]

COMMANDS:
   export   exports prom metrics to csv file
   idle     Find metrics that are not used in any grafana dashboards & prom rules
   help, h  Shows a list of commands or help for one command


```

`owl dashboards --help`
```commandline
NAME:
   owl dashboards

USAGE:
   owl dashboards command [command options]

COMMANDS:
   export    exports grafana dashboards to csv file
   top-used  Lists metrics & rules that are used most in the grafana dashboards
   idle      Find panels in the dashboard whose metrics don't exist anymore'
   help, h   Shows a list of commands or help for one command

```