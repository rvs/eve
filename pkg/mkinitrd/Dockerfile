FROM alpine:3.10 as initrd

COPY nlplug-findfs.c initramfs-init /tmp/
RUN apk add --no-cache mkinitfs gcc musl-dev linux-headers kmod-dev util-linux-dev cryptsetup1-dev
RUN cc -Wall -Werror -g -D_GNU_SOURCE -DDEBUG -I/usr/include/blkid -I/usr/include/uuid /tmp/nlplug-findfs.c -lblkid  -lkmod  -L/lib -lcryptsetup -o /sbin/nlplug-findfs
RUN mkinitfs -n -i /tmp/initramfs-init -o /initrd.gz

FROM alpine:3.10 as build

WORKDIR /out
COPY make-raw install /out/

RUN mkdir -p /out/etc/apk /out/boot && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
  mtools \
  dosfstools \
  libarchive-tools \
  sgdisk \
  e2fsprogs \
  util-linux \
  squashfs-tools \
  coreutils
RUN echo "mtools_skip_check=1" >> etc/mtools.conf

RUN find . -print -depth | cpio -o -H newc | gzip > /installer.gz

FROM alpine:3.10

COPY --from=build /installer.gz /
COPY --from=initrd /initrd.gz /
RUN gzip -d < /installer.gz | cpio -id 2>/dev/null
RUN mkdir /tmp/r ; cd /tmp/r ;\
    for i in initrd.gz installer.gz ; do gzip -d < /$i | cpio -id ; done ;\
    find . | sort | cpio --quiet -o -H newc | gzip > /initrd.img

COPY grub.cfg /EFI/BOOT/

ENTRYPOINT [ "/make-raw" ]
