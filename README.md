# Personal Email Assistant

This project lets you set up your own personal email assistant, who handles relaying incoming email messages to you, hearing your questions or replies, and then writing the reply for you on your behalf.
It started as a fun little project where I wanted to pretend I had a fancy 18th century butler who manages my emails for me, and ended up being a good way to accidentally send lots of spam emails to random poor souls who decided to send me an email. Anyway, I figured it would be a fun way to annoy my friends and random newsletter companies by having their messages answered on my behalf by an AI assistant that speaks in a slightly patronizing Victorian English voice.

WIP: I'm also trying to make this tool be able to support customized personalities. If I can get it working well and consistently enough, I might even try to implement auto-reply features where I let the AI handle sending replies without my intervention, based on some criteria. This feature however still needs some work, so not really ready yet.

## Required Setup

If you want to use this for your own purposes, you will need:

1. A gmail account

2. A Google Cloud project, including a credentials JSON for the Gmail API (instructions to follow)

3. Ollama installed on your computer, for access to llama3.

Let's dive into these!

### Google Cloud Project Setup

So, the main thing you need to obtain here is an Oauth credentials JSON for the Gmail API. This is actually pretty straightforward, so no worries.

1. Go to the Google Cloud console and create a project. Name it whatever you like.

2. Enable the Gmail API:

-   Once your project is selected, go to the Gmail API page. You can find a tile that links to the API page from the console, and search for Gmail from there.
-   Click "Enable" to activate the Gmail API for your project.

4. Set Up OAuth 2.0 Credentials:

-   Navigate to the Credentials page.
-   Click "Create Credentials" and select "OAuth 2.0 Client ID".
-   You will be prompted to configure the OAuth consent screen. Follow the instructions to set up the consent screen. This typically involves providing basic information such as the application name and your email address.
-   Once the consent screen is configured, proceed to create the OAuth 2.0 Client ID.

5. Create OAuth 2.0 Client ID:

-   Choose "Desktop app".
-   Provide a name for the client ID (e.g., "email-assistant").
-   Click "Create".
-   A dialog will appear with your newly created credentials. Click "Download" to save the credentials.json file to your local machine.

### Ollama and llama3

This application requires you to have the `ollama` cli tool installed on your computer - which gives you access to all sorts of LLMs, including llama3, and lets you run them locally on your computer. The best part about it? It's free (unlike using an API from a company such as OpenAI).

- follow the instructions for installing the `ollama` cli tool to your computer. It should be right on the main page:  https://ollama.com/
- this is the LLM you will be downloading:  https://ollama.com/library/llama3

I think as long as you have `ollama` installed and callable from the terminal, then this application will work fine. But I'd recommend you also start by getting `llama3` fully downloaded, since that may take several minutes (it's 4gb, I think).
If you run a command like `ollama run llama3`, I believe it will start the download for you, and then start an interactive chat with the AI. Once the download is done, you can close out of there.

TODO: just make this process automated via some setup script?

## Code Dependencies

You need Go installed on your computer to run this application. Go's main page has easy instructions to install, if you don't have it yet.

## Usage

This application will run in a terminal indefinitely, checking for new emails every hour (TODO: configurable?). When a new email is found that the AI thinks deserves a response, it will begin a dialog with you in the terminal, relaying its contents to you and asking how you'd like to respond. You can tell it how you want to respond, and it should create an message based on what you told it.

### Configuration Options

You will can customize the `config.json` file to set the following properties:

```json
{
    "user_name": "Ben Webb",
    "personality_id": "valet_01", // the .json file holding the personality the AI will use
    "gmail_address": "ben.webb340@gmail.com",
    "inbox_check_freq": 60, // how frequently your gmail inbox will be checked
    "email_batch_limit": 5, // limit on how many emails the AI will bring to you for a reply
    "lookback_days": 10, // limit on number of days back to look for emails
    "debug": true
}
```
