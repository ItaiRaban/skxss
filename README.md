# skxss
Sleep kxss

Added delay option and headers adding to [@tomnomnom](https://github.com/tomnomnom/)'s great tool, and used the adaption of [Emoe](https://github.com/Emoe). 


All the credits goes to:

[@tomnomnom](https://github.com/tomnomnom/) - Creator of [kxss](https://github.com/tomnomnom/hacks/tree/master/kxss)

[@Emoe](https://github.com/Emoe) - Creator of [kxss Adaption](https://github.com/Emoe/kxss)


# Installation

```go get github.com/ItaiRaban/skxss```


# Usage

```cat domain.txt | skxss -d [Millieconds] -h "<HeaderName>: <HeaderContent>" -h "<SecondHeaderName>: <SecondHeaderContent>"```

For example:

```cat domain.txt | skxss -d 2000``` - will wait 2 seconds between each url's scan, without any additional headers

```cat domain.txt | skxss -d 500 -h "UserAgent: MyUserAgent"``` - will wait half a second between each url's scan, and add the header ```UserAgent: MyUserAgent```

Default delay is 0 (Without ```-d``` flag)

# Any Problems?

Feel free to send me a massege on [Twitter](https://twitter.com/itairaban)
