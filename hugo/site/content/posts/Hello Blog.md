---
title: "Hello Blog"
date: 2019-06-27T20:49:13-04:00
draft: false
---

Today, I decided to create a website for myself, to host random code and blog about (mostly) programming.

# Tech Stack
To build this blog, I used [Hugo](https://gohugo.io/) with the [Binario](https://github.com/vimux/binario) theme. To host this, I ran Hugo and a [custom reverse proxy](https://github.com/jadr2ddude/website/blob/master/jprox/main.go) with docker-compose on a VM on Google Cloud Platform.
I am using Cloudflare for DNS and HTTPS.

In the past, I have used a lot of web hosting tech, some of which worked out well and some of which did not.
For this blog, I tried to take everything that worked and put it together.

## Hosting
In the past, I have used Google Cloud Platform, Amazon Web Services, and Linode for hosting. I originally started out with GCP, but ended up moving off of it because it did not allow sending outbound mail, and I wanted to run a mail server. After that, I hosted a website and mail server on Amazon Web Services, which was my worst administration experience so far. For starters, the "free trial" did not actually end up free - I was paying a non-negligible amount for disk bandwidth. Additionally, email hosting is hell. The VM available for the free trial of AWS did not have enough memory to run ClamAV (virus scanner) consistently, and I consistently got a spew of Out-Of-Memory errors. Additionally, the server was under heavy load constantly. Not from HTTP requests. Not from email. Rather, the load was coming from SSH. I had to deal with the usual barrage of login-attempt spammers - except there were enough login attempts that they drove CPU usage up to about 50%. When I first discovered this, I quickly turned password authentication off and switched to public-key auth. This did not actually solve the problem - the login spammers kept trying passwords despite the automatic rejection, but it lowered CPU usage somewhat. Next, I set up fail2ban - a piece of software which can read log files from SSH (or other software) and block IP addresses that spam login attempts. Almost immediately after setting up fail2ban, I locked myself out by accident. I was trying to set up access so another person could edit the site, but I messed up and it blocked them. Except we were both on the same network, behind the same NAT. An hour later, the blocking timed out and I managed to get the login setup correct. Eventually, I tried moving to Linode with a larger server. On Linode, I continued to fight with my mail server (postfix), and we decided to move to G Suite w/ gmail for email hosting (since this was a non-profit, and it was free). At this point, to minimize costs, I moved hosting to my desktop. This block of text is big enough, so I will talk more about local hosting when I get to Cloudflare.

I really have not had any strong negative experiences with GCP, so that is what I chose to go with.

For these domains, I have not yet figured out email hosting. I will probably write another blog about email once I figure it out.

## Site Generation
For the first site I hosted, I was given a folder with all plain HTML files that had been copied and pasted together. To start, I had just used these files and made any edits to everywhere applicable. But I eventually found this annoying when making changes to the footer of all pages, and made a shell script to put together the page from seperate HTML files. This went through many iterations, using makefiles, Lua, and even a Go script with `text/template`. All of these ended up being really complex to manage, and tended to break unexpectedly.

For my personal site, I decided to take the easy way out and use Hugo.

## Web Server
When I first started, I used nginx because it is fairly popular and there is plenty of good documentation. For HTTPS, I used Let's Encrypt which worked with nginx fairly well. However, in part due to my lack of understanding, my nginx configs slowly grew more complex until I could no longer easily understand them. During one of my migrations, I decided to move over to a setup with Caddy and Gogs. I used the git extension for Caddy to automatically retrieve a copy of my site and invoke the site generation script. I also was able to host a testing branch on a subdomain in a similar way.

Later on, I started running into progressively weirder problems with this. With HTML5 server-sent-events, I was getting weird arbitrary delays, which turned out to be a result of buffering inside of Caddy's proxy mechanism, for which there was no straightforward solution. Also when heavily modifying my Caddyfile without storing TLS certs, I got rate limited by Let's Encrypt (mostly my fault). Weirdest of all, I found out that a bug in git was causing zombie processes to leak when I ran Caddy with the git extension in a docker container when there was a network problem (or GitHub/Gogs was down, as occurred on multiple occasions).

This time, instead of worrying about configuration weirdness, I wrote a simple specific-purpose reverse proxy from scratch in Go. Since I wrote and understand every line, I doubt I will run into any significant incomprehensible behavior, and if I do it will be easier to debug.

## Docker & Compose
Through the evolution of my first site, I developed a gnarly set of shell scripts. One of these spun up docker containers in waves - first wave had no dependencies, second depended on stuff in the first, etc., using `&` and `wait` to do this in parallel. Another tore everything down, simultaneously stopping all containers, then simultaneously deleting all containers. Sometimes the scripts worked fine and I could continue working without a problem. However, a fair amount of the time the scripts did one of the following instead:

1. broke the docker daemon (trust me, you do not want to know more)
2. caused the host machine to grind to a halt for ~15 minutes
3. spent forever pulling everything because I accidentally ran `docker image prune` after running the tear down script
4. caused excessive downtime due to restart time
5. one service failed to go up, causing a crashing domino effect
6. timed out systemd and got killed


For this reason, I am now using docker-compose.

## Cloudflare and IPv6
The problem most people have with IPv6 is that they don't have a network supporting it. I had the opposite problem - when locally hosting, the only address I could expose ports on was IPv6. In order to allow IPv4 users to access my site, I used Cloudflare. However, Cloudflare has solved far more than just IPv4 support. Cloudflare took care of TLS, DNS, and analytics.

I already had Cloudflare more or less set up, and was used to it, so I decided to use it for this site.

## The Repo
I put everything for this site (except my TLS certs) in this repo on GitHub:
https://github.com/jadr2ddude/website

The `./up.sh` script starts it, but you will need TLS certificates in order to run it. For local testing of the static site, I just use `cd hugo/site && hugo server`.

# What is going on this site?
* Talk about whatever coding projects I am working on or have built
* Hosting for projects that are exposed via HTTP
* Rants maybe?
* I don't know


Congratulations! You have made it through an extremely boring blog post!
