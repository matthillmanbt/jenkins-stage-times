# Jenkins Stage Times

Simple CLI utility to show the average stage times for the most recent successful Jenkins builds in a given pipeline.

To use, set the following env vars:
```
export JENKINS_HOST=https://<host>
export JENKINS_USER=<jenkins user>
export JENKINS_KEY=<api key for that jenkins user>
```

Can optionally pass the pipeline name (defaults to "master")
