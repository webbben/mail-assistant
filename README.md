# Personal Email Assistant

This project lets you set up your own personal email assistant, who handles relaying incoming email messages to you, hearing your questions or replies, and then writing the reply for you on your behalf.
It started as a fun little project where I wanted to pretend I was some kind of 18th century noble that has his own personal assistant - I've been reading too many Alexandre Dumas novels recently, I think. Anyway, I figured it would be funny to annoy my friends and random companies by having their messages answered on my behalf by an AI assistant that speaks whimsical, semi-archaic formal English.

Since then, I decided to start generalizing this tool so that it could be configured to assume other personalities too. And in the future, I have more fun ideas, like telling the AI information about specific subjects, and letting it reply on my behalf if emails about those topics come in. Think of it like an advanced version of an "automated OOO" reply email, but for any topic you want.

## Required Setup

If you want to use this for your own purposes, you will need:

1. A gmail account

2. A Google Cloud project, including a credentials JSON for the Gmail API (instructions to follow)

3. An OpenAI account, and API key.

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

### OpenAI Setup

This was pretty easy when I did it. You basically just go to OpenAI's website, make an account if you don't have one yet, and then go to the API page. From there, it should be pretty easy to find your way to the API key/secret key creation page. Make sure to copy it into a text file and keep it somewhere you won't lose it, since they won't show it to you a second time.

You will need to add some funding to your OpenAI account though - sadly this API isn't completely free. But I think it's like a few cents per 1M tokens generated, or something like that.

## Code Dependencies

You need Go installed on your computer to run this application. Go's main page has easy instructions to install, if you don't have it yet.

## Usage

This application will run in a terminal indefinitely, checking for new emails every hour (TODO: configurable?). When a new email is found that the AI thinks deserves a response, it will begin a dialog with you in the terminal, relaying its contents to you and asking how you'd like to respond. You can tell it how you want to respond, and it should create an message based on what you told it.

### Configuration Options

You will can customize the `config.json` file to set the following properties:

```json
{
    "user_name": "Ben Webb", // your name; the AI will call you this, or a derivative of it, depending on its persona.
    "ai_name": "Francois", // the name your AI will go by; mainly just for display.
    "prompt_id": "valet5", // name of the text file containing the prompt to give to the AI.
    "gmail_address": "ben.webb340@gmail.com", // your Gmail address. Note that other email types don't work, since we use the Gmail API specifically.
    "inbox_check_freq": 60 // interval in number of minutes in which your Gmail inbox will be checked for new emails.
}
```

### Which emails will the AI assistant try to reply to?

Any email that meets the following criteria:

-   email hasn't already been replied to by this application/AI. This includes emails that have been reported to the user and then purposely ignored.
-   email was received in the past day. this prevents the AI from trying to reply to tons of old emails upon starting.
-   TODO email sender is not on the **ignore list**. You can configure the ignore list to purposely ignore specific senders.
-   TODO email subject does not contain an **ignore term**. You can make a list of terms which, if any are present in an emails subject line, will cause the email to be ignored.

Features I want to implement in the future:

-   Spam ignore filter; Have the AI first assess if an email is spam or automated before bothering the user to reply.
-   Autoreply to specific topics; Give the AI a small list of specific topics which, if an email is detected to pertain to it, the AI will reply to it on its own based on a predefined type of response prompt set by the user. For example:
    -   Topic: "Email about my job, or about software engineering related to company X"
    -   Reply: "Tell the sender that Ben is out of the office on vacation, and won't be back until June 20th. If the email is about databases, refer them to reach out to Dave."
    -   If an auto-reply is sent by the AI, it very briefly informs the user about it in the terminal.

### Can the persona of the AI assistant be customized?

Yes, it can! That is as simple as providing a new prompt text file. You can look at the existing prompt files in `/prompts`, but the general idea is you will tell it what persona to assume, and how to speak. I'd recommend leaving the rest of the prompt as close as possible to the original form though, as they can be very finicky. **or**, if you discover a better prompt than what I currently have preset, let me know!

However, you should make sure the prompt text file has the following:

-   a `%s` for adding your name to the prompt. This should be the **first** instance of it in the prompt text.
-   a `%s` for adding the message content that will be read by the AI. This should be the **second** instance of it in the prompt text.

To be able to use your new prompt file, you should put it in the `/prompts` directory.
