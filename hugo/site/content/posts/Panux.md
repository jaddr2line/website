---
title: "The Rise and Fall of Panux"
date: 2019-07-25T21:40:48-07:00
draft: false
---

One of the things I had wanted to do for a long time was to build a Linux distribution that was simple, straightforward, and decently resource efficient. I attempted this, and did not exactly succeed. In the process, I learned a lot, broke many things, and fundamentally changed my understanding of technology, systems, and software engineering. I intend to share some of the lessons and paradoxes I encountered on my journey.

# The Name
Naming is one of the most difficult parts of development. I did not come up with this name - it was suggested to me on Google Plus ([RIP](https://plus.google.com/)). It comes from the Greek prefix `pan-`, meaning everything or all, and the `-ux` suffix used in Unixes and Unix-related names.

# Failed Initial Attempts
When I originally started, I really did not know what I was doing all that well. I tried hacking a collection of shell scripts and makefiles together. None of these attempts panned out all that well, and quickly blew up in complexity. None of these attempts actually got far enough that they were worth discussing. I gave up, and worked on other stuff for a while.

# Go+Docker+YAML+Makefile = Something?
Eventually, the temptation could not be resisted, and I came back to try again. A lot of changes had happened since my previous try - I learned Go, got some experience with docker, and did enough C to get a better understanding of make. This lead me into an entirely different, possibly ridiculous, approach. Except, it worked from the start. Within a day, I had [uploaded my first package build file](https://github.com/panux/packages-main/commit/3e428628a73ab1edded61f0192ab43c3ef490b5b):
```YAML
version: 1.21.1
sources:
- https://www.busybox.net/downloads/binaries/{{.Version}}/busybox-x86_64
script:
- mkdir out/bin
- install -m 0700 src/busybox-x86_64 out/bin/busybox
```

[At the time](https://github.com/panux/package-builder/commit/31ed20b01f35ea8d3f046679bc36a52f9cf9608d), my build system worked more or less as follows:

1. start up an alpine linux (used to build the initial packages to get a self-building system) docker container with the package generator configuration mounted in
2. configuration parsed as YAML
3. various fields run through `text/template` with `.` set to the parsed YAML
4. sources downloaded
5. script used to generate `script.sh` script file
6. build dependencies installed
7. `script.sh` invoked
8. output tarred and stored to directory mounted from host

You might notice that this script does not invoke any build system, or that the URL of the source leads to a pre-compiled, statically linked binary. This script did not yet actually build anything, rather it tested out the general workflow.

After this, I started slowly building out the dependency graph, adding a `libc` (the C library containing `printf`, `malloc`, etc.), a base file system (defining `/bin` and other core directories), and various utilities. At this point, the closest thing to package management was "untar the sources into a directory and turn it into a docker image". As I added more components, I started noticing common patterns, and extending my configuration format. The only reason I was able to make it far on this project is that this format allowed me to continually extend and refactor it as I discovered what worked and what did not.

## Makefiles Calling Makefiles
One of my quickest observations was that almost everything was a Makefile. I quickly caught on, and ended up changing my generator to output Makefiles. Now, instead of having a bunch of shell scrips calling into makefiles, the source fetching and preparation would be part of a Makefile, and the build `script` could just be the body of a make rule - and run the actual build as a sub-make build.

Looking back, I have no clue whether or not this was a good idea. It might not have actually been important enough to have made a major difference in the long run.

# Templated Build Scripts
One of the biggest improvements on the system came in the form of carefully selected and revised template functions. This allowed me to quickly generate boilerplate without compromising readability. These allowed me to do fairly convoluted things in a straightforward way and quickly replicate them. Here is the [script section](https://github.com/panux/packages-main/blob/982f3d1baa0572cd5e10afac6ba8a0e00e47859a/graphics/mesa/pkgen.yaml#L201) from `graphics/mesa/pkgen.yaml`, one of the biggest package generator files I had, which was written fairly far along in the process:
```YAML
script:
- |
  {{extract "mesa" "xz"}}
  (cd mesa && patch -p1 -i ../src/drmdeps.patch)
  (cd mesa && ./autogen.sh)
  find /usr -name '*.la' -delete
  {{configure "mesa" "--prefix=/usr --sysconfdir=/etc --with-dri-driverdir=/usr/lib/xorg/modules/dri --with-dri-drivers=radeon,nouveau,swrast,i915,i965 --with-gallium-drivers=nouveau,virgl,swrast,svga --with-vulcan-drivers=radeon,intel --enable-llvm --disable-asm --disable-xvmc --enable-glx-rts --enable-llvm-shared-libs --with-platforms=x11,drm,wayland --enable-shared-glapi --enable-gbm --disable-glx-tls --disable-nine --enable-dri --enable-glx --enable-osmesa --enable-gles1 --enable-gles2 --enable-egl --enable-texture-float --enable-xa --enable-vdpau --enable-dri3 --enable-gallium-nouveau"}}
  $(MAKE) -C mesa
  $(MAKE) -C mesa DESTDIR=$(shell pwd)/out/mesa install
  {{pkmv "/usr/lib/pkgconfig" "mesa" "mesa-dev"}}
  {{- range $i, $v := .Data.headers}}
  mkdir -p out/{{$i}}/usr/include
  mv out/mesa/usr/include/{{$v}} out/{{$i}}/usr/include
  {{- end}}
  rm -r out/mesa/usr/include
  {{- range $i, $v := .Data.dri}}
  mkdir -p out/{{$i}}/usr/lib/xorg/modules/dri
  mv out/mesa/usr/lib/xorg/modules/dri/{{$v}} out/{{$i}}/usr/lib/xorg/modules/dri
  {{- end}}
  mkdir -p out/swrast-dri/usr/lib/xorg/modules/dri
  mv out/mesa/usr/lib/xorg/modules/dri/*swrast_dri.so out/swrast-dri/usr/lib/xorg/modules/dri
  {{- range $i, $v := .Data.libs}}
  mkdir -p out/{{$i}}/usr/lib
  mv out/mesa/usr/lib/{{$v}} out/{{$i}}/usr/lib
  {{- end}}
```
The mesa package generator config is one of the longest in Panux, but thanks to templates it is not the most complex. The `extract` template function generates a series of shell commands that untar the package from its computed source path (`src/mesa-17.3.6.tar.xz` from `src/$NAME-$VERSION.tar.$COMPRESSION`), and then strip the version from the extracted folder (`mesa-17.3.6` -> `mesa`). This was a convention enforced by almost every package, and required careful handling to deal with some discrepancies in formatting. A `configure` function is used to generate an invocation of the configuration script that uses appropriate platform & architecture-specific flags.

This package generator config is somewhat special in that it generates a ridiculous number of packages from one build - 29 to be precise. These include headers, GPU-specific DRI implementations, and libraries. In order to accommodate weird situations like this, there is a section in the package generator config called `Data` - which allows arbitrary data to be passed to the template. In this case, I am using it to describe the move operations for splitting libraries and headers into their own packages in an organized fashion:
```YAML
data:
  headers:
    gl-headers: GL
    gles-headers: GLES
    gles2-headers: GLES2
    gles3-headers: GLES3
    khr-headers: KHR
    egl-headers: EGL
    gbm-headers: gbm.h
    xa-headers: xa_*
  dri:
    nouveau-dri: nouveau_*
    radeon-dri: radeon_dri.so
    virtio-gpu-dri: virtio_gpu_dri.so
    intel-dri: i9*_dri.so
  libs:
    libglapi: libglapi.*
    libgles1: libGLESv1_CM.*
    libgles2: libGLESv2.*
    libosmesa: libOSMesa.*
    libgl: libGL.*
    libgbm: libgbm.*
    libwayland-egl: libwayland-egl.*
    libegl: libEGL.*
    vdpau-nouveau: vdpau
    libxatracker: libxatracker.*
```

This clearly labeled structure allows me to specify complex package-splitting operations in a way which is clearer than if I had written a giant chain of `mv` commands.

# The Build System Mess
One of the weirder questions involved in the process of building Panux was that of how to actually turn package generator configuration files into code. This became the most difficult part of Panux, exceeding the complexity of counteracting the issues involved in getting a single stubborn package to compile.

## Makefiles
When this originally started, I hacked together some shell scripts and Makefiles to dump everything into docker containers and run builds. In this process, I gradually moved some parts outside of the containers - like downloading dependencies - so that make could cache them. This repeatedly blew up in my face - errors would result in inconsistent states, caching would break, and there would be spurious results. Additionally, some things were incredibly complex to represent in Makefiles.

Eventually, I weeded through most of these problems and built a system of makefiles capable of correctly caching everything most of the time. But there was a problem: make had a hard performance ceiling. This system had to generate and load hundreds of thousands of Make rules, and this meant that it would take 15 minutes after the issue of a build command to set up the minimum state required to start compiling.

### The Exploding Container Daemon Incident
At the time, I was running Arch Linux on my laptop, and Ubuntu on my desktop. My laptop was good enough for basic development, but all of the real package builds had to be run on my desktop due to the slowness of code building. I wrote a tweak to log output from builds using the `&>` operator, which is supposed to redirect both stdout and stderr to a file. I tested this on my laptop, and it seemed to work fine.

I then ran it on my desktop, and my terminal instantly became unresponsive. My disk indicator was at 100%. Arch Linux uses the `bash` shell by default for builds, while Ubuntu uses `dash`. `dash` interprets `&>` as two separate operators - `&` (fork) and `>` (write to file). So when I ran this on Ubuntu, it started the build in the background and then created an empty file.

The final result: docker attempted to start over 10000 containers, each running concurrent make, at once on my desktop.

## Clean Builds
As a result of my horrible initial experiences with my makefiles, I was lead to a policy of clean, reproducible builds. I would run everything in a "clean-room" environment. This meant isolating everything as much as possible, using a custom minimalist docker image for each build, and avoiding usage of the bootstrap system (the alpine container) as much as possible.

## lbuild
As a part of my experience with makefiles, I realized that many of my problems could be solved by using a system that could dynamically unravel the dependency tree. This meant that rules could be generated _as other rules were being run_. To do this, I built a system that was almost entirely implemented in Lua, using the LuaJIT Foreign Function Interface to call the low-level C functions (`fork()`, `exec()`, etc.) I needed. The startup overhead was effectively minimized, and there were massive speedups from the start. However, it would spontaneously stop on certain occasions. I discovered that these were dependency cycles. The fundamental flaw of this system was that it failed to provide any rational failure mode in case of an unresolvable deadlock, due to the use of promises for resolving dependencies and executing rules.

### Multiple Terabytes of Docker Images
The final breakage for my entire build design came when I ran out of disk space during a build. For each build, I was generating a docker image for the build with all of the dependencies. There were thousands of these gigantic docker images, and I eventually reached a point where the images used in a build could not all be stored at once on my disk.

## High-Performance Build Service w/ Kubernetes
After each build system failed to live up to expectations, my perception of what I was looking for in a build system changed. I was now looking for the following in a build system:

* parallel & concurrent builds
* continuous rebuilds
* no temporary docker images
* fast dependency resolution
* good handling of dependency cycles
* good error handling - build all build-able packages
* handle multiple types of build machines

I ended up deciding on Kubernetes, as it supported scheduling containers across multiple machines with different architectures, and handling automatic service restarts and deploys. Part of this decision was also that I just wanted to learn Kubernetes.

To accomplish this, I built a service that would start out by running a git pull from my package build config repo. It would then parse every package config, and build a dependency graph. The graph would be checked for cycles, and then work would begin running actual builds. For each build, it would first generate some asymmetric keys and TLS keys and put them into a Kubernetes secret. These would be used to provide secure communication between the build pod and the build management service. The secret would be used to create a pod to actually run the build in. The build pod would run a `worker` service, which allowed the build manager to feed it operations to manipulate the local environment by injecting files, running commands, and retrieving outputs. Everything needed to bootstrap a running environment would be shipped in and executed by the build manager in order to prepare for the build. The sources were then downloaded and transferred directly to the container, before finally adding the build scripts and executing the build. After everything that could be built was built, it would wait and then run another git pull and repeat.

### Performance Optimizations
When I first started, there was a ridiculous amount of overhead involved in the build process. I was able to gradually add optimizations and eliminate overhead.

The first problem was that my cycle-detection algorithm was incredibly slow. I was recursing through all dependencies of each build, and allocating tons of tiny structures along the way. Thanks to some help from the `#performance` channel on [Gophers Slack](https://invite.slack.golangbridge.org/), I was able to adopt Tarjan's [Strongly Connected Components Algorithm](https://en.wikipedia.org/wiki/Tarjan%27s_strongly_connected_components_algorithm). After brushing up on graph theory, I got some code set up to extract the strongly connected components of the dependency graph and quickly isolate any cyclical components. After this, the time taken for pre-build setup dropped from 15 minutes to about 3 minutes.

The next bottleneck was involved in my cache management. As my code was set up, caching worked by taking the `SHA-256` hash of all inputs to a build. When I saw that this was becoming a bottleneck, I tried to parallelize it. However, in the end I ended up caching my hashes. I would keep the hash of the data, then check if the last modified date was newer than the last time I hashed it. This meant that for the most part things only had to be hashed once. As a result, my caching of cache hashes was able to dramatically speed up build preparation, reducing it from minutes to seconds.

Also, combinations of errors would sometimes result in pod leaks. Pod leaks are a pain.

### Bottlenecks
At this point, after optimizing, there were two main bottlenecks:

1. Kubernetes overhead
2. autotools

This was something Kubernetes was never meant to do. Kubernetes is advertised as something that can do anything and everything, but that is not exactly true. There is a tiny space of small groups of microservices where Kubernetes can work well, but if you step outside of that, Kubernetes starts fighting back. In my case, my entire build processing system was bottlenecked by Kubernetes, rather than by any actual compilation.

If you don't know what autotools is, you are lucky.

So I determined that the core changes I needed to do were:

1. ditch Kubernetes
2. somehow cut autotools overhead

## Fast Build Tool
I started fixing my problems by taking my Kubernetes service and ripping out Kubernetes. I replaced the `worker` system and pod setup with direct access to the docker API.

### The Docker API Dumpster Fire
The Docker API is one of the worst API designs in Go.

1. __versioning__ - not really a thing; pin a commit
2. __types__ - yup, it has a types package
3. __tar files__ - in Docker, everything is a tar file; if it isn't then you have to make it one; chunked data needs to be buffered entirely in memory
4. __race conditions__ - a container may disappear before you send a delete command
5. __documentation__ - large sections are either undocumented or ambiguously undocumented

### Switching Back to a CLI
During this process, I chose to ditch the CI-type system and instead went with a simpler command-line tool. This gave me greater flexibility to do partial reabuilds.

# Package Management
Package management is a fairly difficult problem. Additionally, since this was supposed to be a tiny system, the package manager had to be tiny. I had bounced back and forth on how to best set up package management. To start, I built a package manager called `lpkg` in Lua. Unfortunately, I never quite settled on any good solution, and ended up with a package manager written in shell script. Each rewrite improved some factors, but messed other things up.

To minimize space, I did downloads over HTTP with busybox by default and then verified downloads with [minisign](https://github.com/jedisct1/minisign). Unfortunately, many iterations of my code ended up with serious problems, including a case when `lpkg` would continue despite a `minisign` verification failure. Also, speed always remained a problem in every version.

# Init
For a while I looked at different init systems, trying to pick one that fit well. Of course, all major init systems are messes. There were three things that I was looking for in an init system:

1. simplicity
2. speed
3. correctness
4. small size
5. effective dependency handling
6. socket-based controlling interface

|Property|SysV Init|SystemD|OpenRC|
|---|---|---|---|
|simplicity|yes|no|yes|
|speed|no|yes|yes|
|correctness|yes|no|yes|
|small size|yes|no|no|
|effective dependency handling|no|yes|yes|
|socket-based controlling interface|no|yes|no|

I looked at more init systems, but I just chose to show these three. Of the existing init systems I looked at, OpenRC was probably the best. However, I ended up deciding to build my own.


![XKCD "standards"](https://imgs.xkcd.com/comics/standards.png)

_from [xkcd](https://xkcd.com/927/)_


As you probbably guessed, [`linit`](https://github.com/panux/linit) did not solve all of the world's init problems. I ended up with something that was essentially a fusion of OpenRC and a C verision of `lbuild`. Additionally, difficult bugs crept up.

The way everything is set up right now, there is no real straightforward way to implement an init system matching all of these criteria. Dependency-based job graphs are difficult to implement to start with, and they become even more difficult when you add constraints on size and usage of a non-concurrent language.

# Vendoring is not Optional
This system worked by downloading all dependencies into a cache. After one fetch, it would be saved permanently. Unfortunately, it had to sometimes be reset, or packages needed to change to different versions. Or, garbage would get sucked into the cache (mostly from SourceForge), and ripping it out would be a pain.

About 50% of the time I did a full from scratch build, something would not be accessible. GitHub would be down, or busybox.net, or some SourceForge mirrors. In my most ridiculous incident, GitHub, busybox.net, and all SourceForge mirrors I was directed to, were all down at once.

What I really should have done was build some sort of vendoring system. I had so many gigantic dependencies that git was not going to cut it, so I actually started building my own system for handling this. This was never completed, as Panux was shut down while it was still in early stages of development.

# It Falls Down
This project was based on the assumption that the build graph could easily be transformed into an acyclical graph. It cannot be. Eventually, holding up this system took tons of time, time which I just did not have. I gradually started to shut down pieces of it, starting with the build system. The website is now down too.

# Was it Worthwhile?
Yes. It was. At this point, I have a deep understanding of what goes into actually building and packaging software. These are all hard problems because there is no perfect solution. There will never be any real perfect solution to any of these problems. Everything has its drawbacks.