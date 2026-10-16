# Inviting people to the app

Delivered at handover by the `deploy-byoa-aws` plugin. It is yours from now on: edit it freely.

If the app signs people in (`services.auth` in `albitor.yaml`), the user pool is invite-only: nobody
can register themselves, so you add each person.

## To invite someone

1. Open this repository on GitHub, then **Actions**, then the **invite** workflow.
2. Choose **Run workflow**, enter the person's email address and run it. You need write access to the
   repository to run a workflow; if the button is missing, ask a repository admin.
3. The person receives an email with their username and a temporary password, valid for 7 days.
4. At their first sign-in the app asks them to choose a password of their own. Nobody else ever
   knows it.

Running the workflow again for someone who has not signed in yet sends a fresh invitation. For
someone who already has, it changes nothing: they can use **Forgot your password?** on the sign-in
page.

## Accounts you did not create

The `dev` deploy also seeds two accounts on `*.invalid` addresses, which receive no mail: a demo
account for trying the app on handover day, and a verifier account the platform's checks sign in
with. Their passwords are held in the app's secret store, not in this repository. Do not invite
anyone to either address.
