# AWS News Feed

A Lambda function that polls the AWS What's New and AWS News Blog RSS feeds and
posts each new item to a Bluesky account.

## Configuration

The Lambda is configured via environment variables:

| Variable | Description |
| --- | --- |
| `BLUESKY_HANDLE` | Bluesky handle to post as (e.g. `awsnews.bsky.social`). |
| `BLUESKY_PASSWORD_PATH` | SSM Parameter Store path containing the Bluesky app password (decrypted at startup). |
| `DYNAMODB_TABLE_NAME` | DynamoDB table that tracks which items have already been posted. |
| `WHATSNEW_RSSFEED_URL` | URL of the AWS What's New RSS feed. |
| `NEWSBLOG_RSSFEED_URL` | URL of the AWS News Blog RSS feed. |
| `LOG_LEVEL` | Optional. zerolog level (`debug`, `info`, `warn`, `error`). Defaults to `info`. |

## Build

```sh
make build      # Lambda artifact (linux/arm64) at bin/awsnewsbot/bootstrap
make package    # zips the artifact at artifacts/awsnewsbot.zip
make deploy     # pushes the zip to AWS Lambda
make local-build # native build for local testing
```

The build embeds the current git describe output as `main.Version` and logs it
at startup so the deployed commit is visible in CloudWatch.

## Behaviour

- Posts are rate-limited to one per second.
- Each invocation processes at most 50 posts so a backlog cannot exceed the
  Lambda timeout; remaining items are picked up on the next scheduled run.
- Each post is created with a deterministic `rkey` derived from the item GUID,
  so a re-attempt after a crash or transient failure cannot create a duplicate
  on Bluesky.
- A new Bluesky session is created on every invocation, which avoids stale
  session JWT failures on warm Lambda containers.
- The two feeds run independently; one feed's failure does not silence the
  other.