# intentd: Agentic cronjobs

This repository implements a terminal user interface to run agentic **Intents**.

An **intent** is not just a *tool*, it is a specific activity that an (L|S|V)LM
agent executes on a repetitive basis, or once, and which is controlled and
guarded by human review processes.

The `intentd` tool features a suite of pre-defined *intents* that may be run
repetitively using your preferred cron/PM system.

## Examples

```bash
# summarize a single link or document
$ intentd summarize https://evi.as/hire-me

# fetch latest entries from a feed and build insights
$ intentd feed https://evi.as/feed/blog.xml 

# explain what is to be seen on a picture
$ intentd vision https://evi.as/assets/evias_with_marley_and_maya.png

# ELI5 on any topic
$ intentd explain https://evias.be/assets/bitcoin.pdf
```

## License

Copyright 2026 Grégory Saive <greg@evi.as> for re:Software S.L. (resoftware.es).

Licensed under the [3-Clause BSD License](./LICENSE).

