FROM golang:1.10

#install petsc
ENV PETSC_VERSION 3.8.3
ENV PETSC_ARCH arch-linux2-c-opt
ENV PETSC_DOWNLOAD_URL ftp://ftp.mcs.anl.gov/pub/petsc/release-snapshots/petsc-lite-$PETSC_VERSION.tar.gz
ENV PETSC_DIR /usr/local/petsc
ENV PETSC_LIB $PETSC_DIR/$PETSC_ARCH/lib/
ENV LD_LIBRARY_PATH $PETSC_LIB:$LD_LIBRARY_PATH
RUN set -eux; \
	cd $PETSC_DIR/..; \
    curl -fsSL "$PETSC_DOWNLOAD_URL" -o petsc.tar.gz; \
	tar -xzf petsc.tar.gz; \
	rm petsc.tar.gz; \
	mv petsc-$PETSC_VERSION petsc; \
	cd $PETSC_DIR; \
    ./configure --with-cc=gcc --with-cxx=0 --with-fc=0 --with-debugging=0 \
      COPTFLAGS='-O3 -march=native -mtune=native' \
      --download-mpich --download-f2cblaslapack; \
    make all test; \
    rm -rf /tmp/* /var/tmp/*;