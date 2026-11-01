# Adding people to the app

Delivered at handover by the `deploy-byoa-aws` plugin. It is yours from now on: edit it freely.

If the app signs people in (`services.auth` in `albitor.yaml`), it is **invite-only** unless you
chose self sign-up (`sign_up: open`) when you planned it. In an invite-only app nobody can register
themselves: you add each person. **The platform sends no sign-in email at all** in this mode: no
invitation, no verification code and no password-reset message. People get their temporary password
from you, in person or over a channel you trust.

## Your own first sign-in

When the app was built, the platform created your account with the address you use there and
showed you a temporary password once. Sign in at the app's address with that address and that
password. The app then asks you to choose a password of your own; nobody else ever knows it. The
temporary password is used for that first sign-in only, and re-deploying the app never changes the
password you chose.

## To add someone

1. Sign in and open **People** in the app. Only members of the user-admin group see it; you, the
   owner, are one.
2. Choose **Add person**, enter their email address and, if the app has roles, their role.
3. The app shows a temporary password **once**. Give it to them, with the app's address, yourself.
4. At their first sign-in they choose a password of their own.

From the same screen you can **reset a password** (a new temporary password is shown once, and the
person chooses another when they sign in) and **disable** or **re-enable** an account. A disabled
person cannot sign in; nothing they created is deleted. To share the work, promote a second
people-admin from the same screen, so the app never depends on one person.

The `invite` workflow in this repository's Actions tab refuses to run in an invite-only app, because
it makes the user pool email the person; use the People screen. It still works if the app lets anyone
register (`sign_up: open`).

## If you are locked out

Anyone who is not the owner asks an admin to reset their password from **People**. If the owner
cannot sign in (a forgotten password, or the account was disabled):

1. On the platform, ask for owner recovery. It shows you a new temporary password once.
2. In this repository open **Actions**, then the **recover owner** workflow, and **Run workflow**.
   You need write access to the repository. It runs under the deploy role, sets that temporary
   password on your account and re-enables it. It prints nothing secret.
3. Sign in with your address and the temporary password, then choose a password of your own.

## Accounts you did not create

The `dev` deploy also seeds accounts on `*.invalid` addresses, which receive no mail: a demo account
for trying the app on handover day, and verifier accounts the platform's checks sign in with. Their
passwords are held in the app's secret store, not in this repository. Do not add anyone at an
`*.invalid` address.

## If the app lets anyone register

With `sign_up: open` anyone with an email address can create an account and confirm it with a code
emailed to them, and there is no People screen. The owner is not seeded in this mode.
