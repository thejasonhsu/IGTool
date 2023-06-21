# instagram-unfollowers
Find out which users are not following you back.

## Initial setup and usage
- Download your information from Instagram
  - Select types of information: `Followers and following`
  - Format: `JSON`
  - Date range: `All time`
- Upload the zip file to Google Drive
  - File access: `Anyone with link`
  - Fetch the file url: `Copy link`
  - Copy the file id  
    The URL format is https://drive.google.com/file/d/xxxyyyzzz/view. In this case `xxxyyyzzz` is the file id...
- Set up your repository
  - Fork the [cecobask/instagram-unfollowers](https://github.com/cecobask/instagram-unfollowers) to your account
  - Enable the `findUnfollowers` workflow ([help](https://docs.github.com/en/actions/using-workflows/disabling-and-enabling-a-workflow))
  - Create a repository secret with name `FILE_ID` and value equal to the Google Drive file id ([help](https://docs.github.com/en/actions/security-guides/encrypted-secrets#creating-encrypted-secrets-for-a-repository))
- Run the `findUnfollowers` workflow ([help](https://docs.github.com/en/actions/using-workflows/manually-running-a-workflow))
  - Shortly, check the console output of the job
  - If any users are not following back, you will see info about them in table format

## Subsequent usage
In the future you may want to find out if any users are not following you back again.  
You can proceed as follows:
- Download your information from Instagram
- Find your previously uploaded file on Google Drive
- Upload a new version by clicking `Manage versions`
- Run the `findUnfollowers` workflow