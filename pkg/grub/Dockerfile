FROM linuxkit/alpine:8b53d842a47fce43464e15f65ee2f68b82542330 AS grub-build

RUN apk add \
  automake \
  make \
  bison \
  gettext \
  flex \
  gcc \
  git \
  libtool \
  libc-dev \
  linux-headers \
  python3 \
  autoconf

# because python is not available
RUN ln -s python3 /usr/bin/python

ENV GRUB_COMMIT=71f9e4ac44142af52c3fc1860436cf9e432bf764
ENV GRUB_REPO=https://git.savannah.gnu.org/git/grub.git

COPY patches/* /patches/
RUN mkdir /grub-lib && \
  set -e && \
  git clone ${GRUB_REPO} && \
  cd grub && \
  git checkout -b grub-build ${GRUB_COMMIT} && \
  for patch in /patches/*.patch; do \
    echo "Applying $patch"; \
    patch -p1 < "$patch"; \
  done && \
  git config --add user.email a@b.c && git config user.name a && git add . && git commit -mPATCH

# list of grub modules that are portable between x86_64 and aarch64
ENV GRUB_MODULES_PORT="part_gpt fat ext2 iso9660 squash4 gzio linux acpi normal cpio crypto disk boot crc64 gpt \
search_disk_uuid search_part_label search_label verify xzio xfs video gfxterm gptprio chain probe reboot        \
regexp smbios part_msdos cat echo test configfile loopback"
ENV GRUB_MODULES_efi_x86_64="multiboot multiboot2 efi_uga efi_gop"
ENV GRUB_MODULES_pc_x86_64="multiboot multiboot2 biosdisk"
ENV GRUB_MODULES_efi_aarch64="xen_boot efi_gop"
ENV GRUB_MODULES_pc_aarch64="xen_boot efi_gop"

WORKDIR /grub
RUN for flavor in efi pc; do                                                                                                \
      git clean -f -d -x                                                                                                  &&\
      ./autogen.sh                                                                                                        &&\
      ./configure --disable-efiemu --libdir=/grub-lib --with-platform="$flavor" CFLAGS="-Os -Wno-unused-value"            &&\
      make -j "$(getconf _NPROCESSORS_ONLN)"                                                                              &&\
      make install                                                                                                        &&\
      platform=`echo /grub-lib/grub/*-"$flavor"` ; eval MODS=\"\$GRUB_MODULES_"$flavor"_`uname -m` ${GRUB_MODULES_PORT}\" &&\
      ./grub-mkimage -O `basename "$platform"` -d "$platform" -o /grub-lib/BOOT."$flavor" -p /EFI/BOOT $MODS               ;\
    done

RUN mv /grub-lib/BOOT.efi /grub-lib/$([ $(uname -m) = aarch64 ] && echo BOOTAA64.EFI || echo BOOTX64.EFI)

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /EFI/BOOT
COPY --from=grub-build /grub-lib/BOOT* ./
COPY rootfs.cfg grub.cfg
