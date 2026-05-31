# gshoot

REMIND

## Authentication

Getting `gshoot` to talk to Google Sheets is challenging, to put it mildly. Don't blame me, I do not work for Google and I did not design this system.

`gshoot` talks to Google Sheets as you, using a _Google Cloud project_ that _you create_. Again, I would like to apologize in advance. This is just incredibly complicated and error-prone.

I recommend these three well-written tutorials:

- https://github.com/openclaw/gogcli/blob/main/docs/quickstart.md
- https://github.com/googleworkspace/cli/blob/main/README.md#manual-oauth-setup-google-cloud-console
- https://ucsb-cs156.github.io/topics/oauth/google_oauth_consent_screen.html

The goal here is something like:

1. Create a new **Google Cloud Project** to contain your OAuth setup.

2. In that project, enable two Google APIs - **Google Drive & Google Sheets**. If your project doesn't enable these two APIs, nothing will work. Ever.

3. Configure the **OAuth Consent Screen**. Pick whatever name/email you want, you will are only human alive who will see this screen. If your Google account is a "Google Workspace" account with a custom domain set this up as **Internal Audience**, otherwise use **External Audience**.

4. If you used **External Audience**, add your email as the only test user. This is required. No test user, no access for you. Is it strange Google doesn't automatically do this for you? I think so too!

5. Create a **Desktop OAuth Client**. Yes, I know that `gshoot` has nothing to do with desktop and this is very confusing. This is just what Google calls this kind of authentication.

6. Download the **OAuth Client Secrets** file from your "Desktop App". Google gives it a simple name like `client_secret_XXXXXXXXXXXX.apps.googleusercontent.com.json`

and finally we get to the part where gshoot comes in:

```sh
$ gshoot auth login --client-secrets client_secret_XXXXXXXXXXXX.apps.googleusercontent.com.json
```
