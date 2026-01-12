# Right2Comment

**Right2Comment** is a browser extension that lets people comment on YouTube videos *outside of YouTube*.

Comments are **not** posted on YouTube. They’re stored on a separate server and associated with a video using its `video_id`.

## Why this exists

YouTube comments are controlled entirely by channel owners. That creates echo chambers, heavy-handed moderation, and the quiet erasure of dissenting but legitimate discussion.

Right2Comment separates **discussion from ownership**.

At the same time, I don't want it to be  an anti-creator tool. The system would be explicitly designed to discourage:
- hate speech
- harassment
- misleading or malicious content

## Design philosophy

The extension is intentionally minimal.

- No accounts
- No tracking
- No personal data collection

It only:
- reads the current video’s `video_id`
- sends simple `GET` requests to fetch comments
- sends simple `POST` requests to submit comments

That’s it.

## Why a browser extension?

Because commenting should be frictionless.

Discussion should happen *where the video already is*, not on a separate website nobody checks. The extension approach keeps the experience lightweight, immediate, and context-aware.

## Firefox add-on

Right2Comment is available on Firefox:

https://addons.mozilla.org/en-US/firefox/addon/right2comment/
