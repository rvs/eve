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

COPY --from=initrd /initrd.gz /
COPY --from=build /installer.gz /
RUN gzip -d < /installer.gz | cpio -id 2>/dev/null
RUN cat initrd.gz installer.gz > initrd.img

ENTRYPOINT [ "/make-raw" ]
