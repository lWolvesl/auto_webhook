# auto_webhook

`GoLang`

- Use webhook to start your own script
- install with binary file
- default port `9922`

### start with name

> you should put 123.sh (eg.) into same path,and token should be gaven in token file

`then`

```http://127.0.0.1:9922/job?job=123&token=xxx```

`will run 123.sh and back a id to you`

### kill job by id

```http://127.0.0.1:9922/kill?id=1&token=xxx```

