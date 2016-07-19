# mrpush
irc bot for building jenkins jobs

### usage
configure your jenkins jobs in config.json

pass in the config.json file location as a command line argument

`./bin/mrpush -config=/path/to/config.json`

`!list - returns available jobs`

`!build jobname gitref - submits build to jenkins`

`!status jobname - returns current build status`
