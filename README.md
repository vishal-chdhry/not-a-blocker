# Not a blocker

Not a blocker is a [tflint](https://github.com/terraform-linters/tflint) plugin that finds minor nits in your PR and blocks it from getting merged.

![xkcd-code-quality](https://imgs.xkcd.com/comics/code_quality_3_2x.png)
xkcd: https://xkcd.com/1833/

## Usage

### Requirements
1. Docker

### Steps
1. Add this to the `.tflint.hcl` file at the root of your repository

    ```hcl
    plugin "terraform" {
      enabled = false
      preset  = "recommended"
    }

    plugin "not-a-blocker" {
      enabled = true
      version = "<LATEST_RELEASE_TAG>" # without `v` as prefix
      source  = "github.com/vishal-chdhry/not-a-blocker"
    }
    ```
2. Run the following command at the root of your repository

    ```sh
    docker run --rm -v $(pwd):/data -t --entrypoint /bin/sh ghcr.io/terraform-linters/tflint -c "tflint --init && tflint --recursive --chdir=images --config=/data/.tflint.hcl"
    ```

### Troubleshooting
TFlint provides a environment variable `TFLINT_LOG` to enable logging, set its value to `DEBUG`, `INFO`, `ERROR` in the docker command as follows

```sh
docker run --rm -e TFLINT_LOG="INFO" -v $(pwd):/data -t --entrypoint /bin/sh ghcr.io/terraform-linters/tflint -c "tflint --init && tflint --recursive --chdir=images --config=/data/.tflint.hcl"
```
