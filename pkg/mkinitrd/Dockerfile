FROM alpine:3.10 as initrd

RUN apk add --no-cache mkinitfs
RUN mkinitfs -n -o /initrd.gz

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
