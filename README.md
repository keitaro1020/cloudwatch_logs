Cloudwatch Logs からログを取得する。

## 使い方
```shell
$ go run main.go --profile <aws sso profile> --log-group <log group name> --filter <filter condition> --start <start time [yyyymmddhhmmss]> --end <end time [yyyymmddhhmmss]>
```