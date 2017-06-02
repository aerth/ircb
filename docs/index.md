# ircb

### definition system

Usage:

 * !define [word] [definition goes here]
 * sending `!word` will make ircb reply with definition
 * public command, can be (un)locked with `@set define on|off`
 * definitions are limited to 512 bytes (probably smaller)

Data:

 * stored in database

### karma system

Usage:

 * increment by one: user+, user++, user+++ etc
 * decrement by one: user-, user--, user--- etc
 * increment by one: `user: <text>thank<text>`
 * show self karma: `!karma`
 * show bob's karma: `!karma bob`
 * show top karma user `!karma ^`

### history system

Usage:

 * reply with latest timestamp and message from 'user': `!seen user`
 * search history for latest occurance of 'word': `!history word`
 * master commands: `@clear` `@set history on|off`

### http system

  * master commands: `@set links on|off`
  * will respond to messages with 'http'
  * only replies for valid URL that resolves
  * shows http status code (such as 200, 404), and response time
  * tries to detect content-type
  * no proxy support yet (soon)
  * only downloads small portion of file (useful for large downloads)

### config system

  * json for now


### built-ins

unless you create your own CommandMap and MasterMap, these will be public commands:

  * uptime
  * history
  * karma
  * q (quiet)
  * up
  * help
  * about
  * echo
  * define

and these will be master commands
  * q (quit)
  * do (raw IRC)
  * set [thing] on|off

