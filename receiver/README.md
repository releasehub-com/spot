# Receiver

Webserver running in a cluster that listens for incoming webhooks. This is the entry point into the kubernetes cluster. The webhook is *not* the original webhook received from a Version control system like Github but rather a preprocessed payload from Release that includes more information.
