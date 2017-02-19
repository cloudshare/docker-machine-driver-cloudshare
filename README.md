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

### VM Templates

CloudShare maintains a few VM templates that are docker-machine ready.

![Docker-Ready Templates](https://i.imgur.com/O4d0F5n.png)

By default, the `Docker - Ubuntu 14.04 Server - SMALL` template is used. If you need a bigger VM (more CPU, more RAM) you can specify a different VM template by name.

e.g.

```
docker-machine create \
   --driver cloudshare \
   --cloudshare-vm-template-name 'Docker - Ubuntu 14.04 Server - LARGE' \
   my-machine
```

This is the fastest way to provision a new Docker-Machine - by selecting one of the predefined templates (Small, Medium or Large). If you need more fine grained control on the VM hardware, you can use the following command line options, but **be aware** it takes a few more minutes to launch VMs with customized hardware.

```
   --cloudshare-cpus "1"                    CPU count
   --cloudshare-disk-gb "10"                Disk size (GB), >=10GB
   --cloudshare-ram-mb "2048"               RAM (MBs) 256-32768
```

You can also specify the VM template by ID. You can use the API to obtain a [list of all the templates](http://docs.cloudshare.com/rest-api/v3/environments/templates/get-templates/) and their IDs programmatically.


### More help

Run `docker-machine create -d cloudshare --help` to see all the options.

Make sure you have the CloudShare driver installed first.


## Install

- Grab the [latest release](https://github.com/cloudshare/docker-machine-driver-cloudshare/releases) binary for your OS
- Untar
- Make sure the `docker-machine-driver-cloudshare` executable is in your `PATH`.
    - On Linux (64 bit) the following bash script will do the above for you:
```
export LATEST=$(curl -s -L https://api.github.com/repos/cloudshare/docker-machine-driver-cloudshare/releases/latest | grep tag_name | grep -E "[.0-9]+" -o) && \
    curl -s -L https://github.com/cloudshare/docker-machine-driver-cloudshare/releases/download/${LATEST}/docker-machine-driver-cloudshare_amd64-linux.tar.gz -o /tmp/docker-machine-driver-cloudshare.tar.gz && \
    cd /tmp && tar xf /tmp/docker-machine-driver-cloudshare.tar.gz && \
    mv /tmp/docker-machine-driver-cloudshare /usr/local/bin/ && \
    chmod +x /usr/local/bin/docker-machine-driver-cloudshare
```
- Grab your API ID and Key from your [user details page](https://use.cloudshare.com/Ent/Vendor/UserDetails.aspx)
    - Define them as environment variables: `CLOUDSHARE_API_KEY` and `CLOUDSHARE_API_ID`.
    - You can also pass them directly to `docker-machine` with these options:
        - `--cloudshare-api-id`
        - `--cloudshare-api-key`


