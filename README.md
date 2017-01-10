# CloudShare Docker-Machine Driver

With CloudShare's docker-machine driver installed, you can launch Docker-ready VMs on CloudShare.

## Usage

### Create a docker-machine on CloudShare

`docker-machine create -d cloudshare my-machine`

### Run your docker containers on the new machine

```
eval $(docker-machine env my-machine)
docker run busy-box echo "Look at me, I'm running on CloudShare"
```

### More help

Run `docker-machin create -d cloudshare --help` to see all the options.

Make sure you have the CloudShare driver installed first.


## Install

- Grab the [latest release](https://github.com/cloudshare/docker-machine-driver-cloudshare/releases) binary for your OS
- Untar 
- Make sure the `docker-machine-driver-cloudshare` executable is in your `PATH`.
- Grab your API ID and Key from your [user details page](https://use.cloudshare.com/Ent/Vendor/UserDetails.aspx)
    - Define them as environment variables: `CLOUDSHARE_API_KEY` and `CLOUDSHARE_API_ID`.
    - You can also pass them directly to `docker-machine` with these options:
        - `--cloudshare-api-id`
        - `--cloudshare-api-key`


