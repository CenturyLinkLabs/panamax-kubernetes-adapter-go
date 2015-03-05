![Panamax - Docker Management for Humans](http://panamax.ca.tier3.io/panamax_ui_wiki_screens/panamax_logo-title.png)

[Panamax](http://panamax.io) is a containerized app creator with an open-source app marketplace hosted in GitHub. Panamax provides a friendly interface for users of Docker, Fleet & CoreOS. With Panamax, you can easily create, share, and deploy any containerized app no matter how complex it might be. Learn more at [Panamax.io](http://panamax.io) or browse the [Panamax Wiki](https://github.com/CenturyLinkLabs/panamax-ui/wiki).

###Create Panamax Adapter in Go

Building a remote adapter was designed so teams could use their favorite language, but all of the adapters had been written in Ruby. This article will explain how to create a working Panamax remote adapter using Go.

Everything you need to create and adapter is outlined in the
[Adapter Developer Guide](https://github.com/CenturyLinkLabs/panamax-ui/wiki/Adapter-Developer's-Guide).

This process has been greatly simplified for Go developers by using the pmxadapter project.

#### Build Steps

1. Create your go project
2. Go get github.com/CenturyLinkLabs.com/pmxadapter
3. Implement the adapter interface.
4. Create the server

#### Sample
A sample can be found here: [https://github.com/CenturyLinkLabs/sample-go-adapter](https://github.com/CenturyLinkLabs/sample-go-adapter)

