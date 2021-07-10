# Deploying EVE-OS

Deploying EVE-OS is similar to deploying any regular operating system. It can be installed directly 
on physical hardware (i.e., EVE-OS can run on bare metal) or it can be deployed within a virtual environment (i.e., EVE-OS can run inside of a
virtual machine). When running in a virtual environment, EVE-OS needs to
have access to [nested virtualization](https://en.wikipedia.org/wiki/Virtualization#Nested_virtualization).
While it is possible to run EVE-OS in a virtual environment without nested virtualization
enabled, unfortunately a lot of capabilities will be degraded (e.g., its ability to run accelerated VMs).

## Deployment methods: live vs installer images of EVE-OS

Since EVE-OS follows a very traditional boot process, its *live* image has to be
available on an accessible storage device so that it can be launched by the BIOS (or other bootloader). After being
deployed, EVE-OS includes capabilities for managing and updating its own live image. But the initial installation
often requires some sort of bootstrapped approach to write the live image of EVE-OS to the edge node's
storage device.

In some cases, EVE-OS can be written directly to a storage device and then that storage device can be 
attached to the edge node. This applies to both physical and virtual environments. 
For example, a live EVE-OS image can be written to a USB stick and then the edge node can boot off of that USB stick.
In a virtual environment, the virtualization platform can be instructed to 
take a live EVE-OS image and use it as a hard drive. When these approaches are 
impossible or impractical, we can use an "installation process" instead. It is in 
these latter cases where the installer image works well. One example need for the installer 
approach is when an edge node's only storage device is an eMMC module soldered directly to the edge
node's processor board. Another example is when the USB controller is so slow that booting and/or running off of
an external USB stick will simply take too much time. In these cases an installer image can be run on
the edge node itself, since the node has access to write the EVE-OS live image
onto its integrated storage device. This two-step process is why it 
is called an EVE-OS *installer* image and why releases of the EVE-OS distribution include two separate
binaries -- one called "live" and one called "installer".

There's nothing particularly special about the *installer* image of EVE-OS. In fact, under the
hood, it is simply a live image of EVE-OS that first runs a single application: a tiny script 
that writes a live image onto an integrated storage device; one that is otherwise inaccessible outside the edge node. 
The installer image is essentially a live image that writes itself, using this extra script.
Hence, the installer image is only booted once on a given edge node. After the 
installer's image-writing portion of the script is complete, it shuts down the edge node. 
Thereafter, the live image is available on the storage device so there's no need to run the installer anymore.

In general, once deployed, EVE-OS assumes a hands-free environment that does not rely on a human
operator. No one is required to configure initial settings "at the console". Any 
ncessary configuration can be included as part of the EVE-OS image. 
To find out more regarding how EVE-OS can be configured, check out the 
[configuration](CONFIG.md) documentation. The remainder of this document will
assume that either the config partition of the EVE-OS image has the required configuration information, 
or that dynamic configuration will be done by supplying overrides during the boot process of the installer image. 
Once it is deployed, the assumption is that all application management and monitoring of 
EVE-OS will occur via API calls between the edge node and a remote application called an EVE controller.

## Unique identification: serial vs. soft serial numbers

EVE controller has to be able to recognize EVE for the first time. This is done
by relying on a piece of semi-unique information called a ```serial number```.
The problem with serial numbers is that they come in many flavors and in cases
where you don't have physical access to the edge node you may not even know
the serial number ahead of time. This is why EVE-OS has a secondary serial
number called ```soft serial```. Whenever you run EVE-OS installer image you
can either tell it directly what do you want your soft serial number to be by
passing an alpha-numeric sring in the ```eve_soft_serial``` variable OR you
can rely on the installer image to generate a unique one for you. In the later
case the only problem is getting that number back. EVE-OS installer image
always prints a soft serial number at the end of the run and if the image
is running from a writable media (like a USB stick) it will also deposit
it in the INVENTORY partition as a newly created folder (the folder name is
the soft serial number). 

## Deploying EVE-OS in physical environments

Physical environments assume an actual, physical edge node, but do not necessarily
assume physical access to the edge node. In cases where physical access is not available
it is common to rely on either iPXE booting or remote management solutions such
as [HPE iLO](https://en.wikipedia.org/wiki/HP_Integrated_Lights-Out) or [Dell DRAC](https://en.wikipedia.org/wiki/Dell_DRAC). 

With a few exceptions (like a Raspberry Pi or SBCs in general) a physical edge node
has a storage device that can only be accessed by the software running on the edge node
itself. The way to do this is to run EVE-OS installer image on the device once and then
rely on EVE-OS itself to manage its own live image. This, in turn, comes down to the
following options: run installer via iPXE or run installer via a boot media.

### Running installer image via iPXE

[iPXE](https://en.wikipedia.org/wiki/IPXE) is a modern Preboot eXecution Environment and
a boot loader that allows operating system images to be downloaded right at the moment
of booting. It allows for a high degree of flexibility that is applicable to both
traditional datacenter/hardware lab and most of the public cloud providers offering bare metal
severs (with Equinix Metal AKA Packet.net and AWS EC2 Bare Metal Instances being two major
examples). iPXE expects a configuration file and a set of binary artifacts to be available
to it at certain URLs in order to proceed with the boot process. Every release of EVE-OS
(including the master builds) produces an ```ipxe.cfg``` and all the required artifacts on GitHub
in the tagged release assets area. The same set of artifacts can be obtained locally by running
```docker run lfedge/eve installer_net | tar xf -```. Regardless of whether you're using
artifacts published on GitHub or on your local http server, as long as iPXE can successfully
resolve the URLs under which they are published the process will work. The default ```ipxe.cfg```
file assumes GitHub URLs and therefore needs to be edited if you are deploying in a local
environment. Here's an example of a custome iPXE configuration file used to run an EVE-OS
installer image in a hardware lab (note the changes made to both ```url``` and ```eve_args``` variables:

```console
#!ipxe
# set url https://github.com/lf-edge/eve/releases/download/snapshot/amd64.
set url https://10.0.0.2/eve/releases/download/snapshot/amd64.
# set eve_args eve_soft_serial=${ip} eve_reboot_after_install
set eve_args eve_soft_serial=${ip} eve_install_server=zedcontrol.hummingbird.zededa.net eve_reboot_after_install

# you are not expected to go below this line
set console console=ttyS0 console=ttyS1 console=ttyS2 console=ttyAMA0 console=ttyAMA1 console=tty0
set installer_args root=/initrd.image find_boot=netboot overlaytmpfs fastboot

# you need to be this ^ tall to go beyound this point
kernel ${url}kernel ${eve_args} ${installer_args} ${console} ${platform_tweaks} initrd=amd64.initrd.img initrd=amd64.installer.img initrd=amd64.initrd.bits initrd=amd64.rootfs.img
initrd ${url}initrd.img
initrd ${url}installer.img
initrd ${url}initrd.bits
initrd ${url}rootfs.img
boot
```

Most of the time you wouldn't need to edit ```ipxe.cfg``` since the default values provide
adequate out-of-the-box behavior of the installer. In fact, you can simply supply the URL
of the ```ipxe.cfg``` published on GitHub as the only input to the iPXE executable. For
example, the following command will install EVE-OS on the t1.small.x86 server in the sjc1
Packet.net datacenter (you need to supply your own project ID as XXX and have a functional
configuration under ~/.packet-cli.yaml):

```console
packet-cli -j device create \
           -H eve-installer \
           -p XXXX          \
           -f sjc1          \
           -P t1.small.x86  \
           -o custom_ipxe   \
           -i https://github.com/lf-edge/eve/releases/download/snapshot/amd64.ipxe.efi.ip.cfg
```

With the default configuration, EVE-OS uses its public IP address as a soft serial (if you
want to use MAC address you can switch to ```amd64.ipxe.efi.cfg```). Once the above command
is done creating a server, you will get a server UUID as part of its JSON output. E.g.:

```console
packet-cli -j device create ...
{
  "id": "64031d28-2f75-4754-a3a5-191e50f10098",
  "href": "/metal/v1/devices/64031d28-2f75-4754-a3a5-191e50f10098",
...
```

Keep this UUID handy. You will need it to access a remote serial console of your server:
```ssh [UUID]@sos.sjc1.platformequinix.com``` and also to get its public IP address:
```packet-cli -j device get -i [UUID]```. Note how the host part of the serial console
ssh access has a name of a datacenter ```sjc1``` embedded in it.

### Running installer image from a boot media

Running installer image from a boot media involves:

1. producing a disk-based installer image (by running ```docker run lfedge/eve installer_raw > installer.raw``` command)
2. burning the resulting ```installer.raw``` image file onto a USB stick
3. instructing your BIOS to boot from the USB (don't forget to enable VT-d, VT-x and TPM in BIOS while you are at it)

Since flashing a USB stick by system utilities can be daunting we recommend
GUI based applications like [Etcher](https://www.balena.io/etcher/) for the job.

In rare cases (such as using HPE iLO or Dell DRAC) your BIOS will not be able
to boot from a disk-based image. The fallback here could an ISO image. The only
difference is in step #1 that then becomes ```docker run lfedge/eve installer_iso > installer.iso```

At the end of the run, installer will shut down the edge node and will add a folder
to a the USB stick named after the soft serial number that can be used to oboard
EVE-OS onto the controller.
 
### Deploying EVE-OS on SBCs (including older Raspbery Pi models) and running live

A lot of single-board computers (SBCs), including older models of Raspberry Pi, don't have any kind
of built-in storage devices. They rely on microSD cards that can be put in
as the only storage mechanism. In those cases there's little advantage to running the
EVE-OS installer image since you can write the live image directly to the microSD card. That is, since the microSD card has to
be prepared outside the SBC, putting a live EVE-OS image on it allows you to
skip the entire installer image step. (Remember that the only reason we
have to run an installer image to begin with is because on most edge nodes
you can't just pop the hard drive out to write to it).

The good news, of course, is that there's no real difference between a live
EVE-OS image and installer EVE-OS image. Just as for the installer image you will:

1. produce a live image (by running ```docker run lfedge/eve live > live.raw``` command)
2. burn the resulting ```live.raw``` image file onto a microSD card
3. insert microSD card into your SBC and power it on

Note that exact same procedure would work for regular edge nodes and USB sticks as well.
There is no problem with running EVE-OS from the USB stick (aside from USB controllers
making it painfully slow sometimes).

## Deploying EVE-OS in virtual environments

### Deploying EVE-OS on top of software virtualization providers

EVE-OS is known to run under:

* [qemu](https://www.qemu.org/)
* [VirtualBox](https://www.virtualbox.org/)
* [Parallels](https://www.parallels.com/)
* [VMWare Fusion](https://www.vmware.com/products/fusion.html)

You need to consult EVE-OS's Makefile for the right invocation of these tools.

### Deploying EVE-OS as a VM in public cloud

EVE-OS is known to run on Google's GCP as a Virtual Machine. You need to consult EVE-OS's Makefile for the right invocation of these tools.
