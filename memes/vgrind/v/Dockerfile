from ubuntu:bionic
run apt-get update && apt-get install -y --no-install-recommends gcc make libcurl3-dev binutils libc6-dev valgrind && rm -rf /var/lib/apt/lists/*

run useradd -ms /bin/bash v
user v
workdir /home/v

copy ./v.tar.gz /home/v/v.tar.gz
run mkdir /home/v/v && tar -xf /home/v/v.tar.gz --strip 1 -C /home/v/v

run cd /home/v/v/compiler && cc -std=gnu11 -w -o v-bootstrap v-bootstrap.c && ./v-bootstrap -o v .

user root
run cp /home/v/v/compiler/v /bin/v && chown root /bin/v && chmod 0755 /bin/v
user v

workdir /home/v
add ./runv.sh /home/v/runv.sh
add ./meta.txt /home/v/meta.txt
entrypoint ["./runv.sh"]
