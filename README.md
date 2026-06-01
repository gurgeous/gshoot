[![test](https://github.com/gurgeous/gshoot/actions/workflows/ci.yml/badge.svg)](https://github.com/gurgeous/gshoot/actions/workflows/ci.yml)

<img src="./logo.svg" width="60%">

# gshoot

Magically upload/download CSVs from Google Sheets.

## Installation

On MacOS use brew:

```
$ brew install gurgeous/tap/gshoot
```

For Linux, see the [latest release on github](https://github.com/gurgeous/gshoot/releases/latest). You'll find MacOS builds in there too, but they are difficult to run since they are unsigned. Windows is not yet supported.

### Important Features

REMIND

### Options

```
$ gshoot --help

Magically upload/download CSVs from Google Sheets.

Commands:
  auth login     Login via OAuth. (start here!)
  auth logout    Logout of OAuth.
  auth status    Show auth status.
  down           Download a Google Sheet as CSV.
  up             Upload a CSV to Google Sheets.
  list           List your Google Sheets.
  peek           List sheets in a spreadsheet.
  wipe           Wipe/delete all data from a spreadsheet.
```

## Authentication

Getting `gshoot` to talk to Google Sheets is challenging, to put it mildly. Don't blame me, I do not work for Google and I did not design this system. `gshoot` talks to Google Sheets as you, using a _Google Cloud project_ that _you create_. Again, I would like to apologize in advance. This is just incredibly complicated and error-prone.

I recommend these three well-written tutorials:

- [gogcli](https://github.com/openclaw/gogcli/blob/main/docs/quickstart.md#2-get-an-oauth-client)
- [gws](https://github.com/googleworkspace/cli/blob/main/README.md#manual-oauth-setup-google-cloud-console)
- [UCSB CS156](https://ucsb-cs156.github.io/topics/oauth/google_oauth_consent_screen.html)

The goal here is something like:

| Step                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                | Helpful Screenshot_Had_To_Use_Long_Name_Here                                                                                                                                                                                                                                 |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1. Create a new **Google Cloud Project** to contain your OAuth setup. SELECT YOUR PROJECT!! You might have to wait a second before you can do that, Google is slow.                                                                                                                                                                                                                                                                                                                                 | <img width="543" height="473" alt="image" src="https://github.com/user-attachments/assets/fddca4c0-acb9-40d0-9072-b80724deceeb" />                                                                                                                                           |
| 2. In your new project, enable these two Google APIs - **Google Drive & Google Sheets**. If your project doesn't enable these two APIs, nothing will work. Ever.                                                                                                                                                                                                                                                                                                                                    | <img width="381" height="265" alt="image" src="https://github.com/user-attachments/assets/5501aeb8-204a-4a62-9574-c3a0ea45f90a" />                                                                                                                                           |
| 3. Configure the **OAuth Consent Screen**. Pick whatever name/email you want, you will are only human alive who will see this screen. If your Google account is a "Google Workspace" account with a custom domain set this up as **Internal Audience**, otherwise use **External Audience**. If you use **External Audience**, add your email as the sole test user. This is required. No test user, no access for you. Is it strange Google doesn't automatically do this for you? I think so too! | <img width="535" height="430" alt="image" src="https://github.com/user-attachments/assets/74cc4456-c081-4113-b070-6a0e675fa107" /><br><br><img width="413" height="383" alt="image" src="https://github.com/user-attachments/assets/7ef82e92-930d-4398-ba63-0331745cebe0" /> |
| 4. Create a **Desktop OAuth Client**. Yes, I know that `gshoot` has nothing to do with desktop and this is very confusing. This is just what Google calls this kind of authentication.                                                                                                                                                                                                                                                                                                              | <img width="315" height="410" alt="image" src="https://github.com/user-attachments/assets/40bc40b8-1070-488d-89b5-951a21580e00" />                                                                                                                                           |
| 5. Download the **OAuth Client Secrets JSON** file from your "Desktop App". Google gives it a real simple name like `client_secret_XXXXXXXXXXXX.com.json`                                                                                                                                                                                                                                                                                                                                           | <img width="247" height="541" alt="image" src="https://github.com/user-attachments/assets/3eccd6e9-5594-41c0-a473-2bca72e8a4ca" />                                                                                                                                           |

and finally we get to the part where gshoot can actually do something:

```sh
$ gshoot auth login --client-secrets client_secret_XXXXXXXXXXXX.apps.googleusercontent.com.json
```

### Changelog

REMIND

# TODO

- README
- badges
- vhs demo / screenshots
