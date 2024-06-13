## Personality Configuration

This is where personality files can be added, to configure how the AI behaves.
To create a new personality, you can use the personality setup command line tool:

```shell
go build cmd/personality_setup/personality_setup.go
./personality_setup
```

Once you've made a personality, you can make edits to it directly in the generated json file, which should be saved under `/personality/<personality-id>.json`

Note that in order for this personality file to be usable in this applicaiton, it needs to stay under the `/personality` directory.

### Adding custom terms to prompts

You can optionally define custom terms to insert into your prompts. In the personality json file, `insert-dict` can be used to map key/value pairs.
Then, when prompts are loaded to be passed to the LLM, these values will be inserted wherever the key is found.

For example, if you have the following:

```json
{
    //...
    "insert-dict": {
        "dog-name": "Sparky",
        "company-name": "Balboni Construction"
    }
}
```

Then in a prompt definition, you can add `<<DOG-NAME>>` and `<<COMPANY-NAME>>`, and they will be replaced with their respective values:

```
Hi all,

The woodchipper is stuck again. Can someone please fix that damn thing? We need it for the job tomorrow morning!

Also, has anyone seen a dog on the job-site? I think it's a coconut corn husky. Owner says his name is Sparky.

Tommy,
Balboni Construction
```

**Note:** the keys `user-name` and `ai-name` are reserved, and set by the app configuration and ai name set in the personality file. So the tags `<<USER-NAME>>` and `<<AI-NAME>>` will automatically be set according to that configuration.
