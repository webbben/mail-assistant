### Personality Configuration

This is where personality files can be added, to configure how the AI behaves.
To create a new personality, use the personality setup wizard command line tool:

```shell
go build cmd/personality_setup/personality_setup.go
./personality_setup
```

Once you've made a personality json, you may find you want to modify the prompts and their generated phrase sets.
To do this, you can edit the json directly, and then you can run the same command line tool, with the `rebuild` flag set to the `id` of your personality/filename of the personality file.

If you have a personality json called "persona_01.json", you would enter:

```shell
./personality_setup -rebuild=persona_01
```

Note that the tool expects to find this file under the `/personality` directory.
