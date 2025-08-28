# CPDLC Logon & Request Example

This example shows the process of setting up 2 goroutines. One to listen to incoming ACARS messages. The other, to handle the on connected event to then process any further requests like the climb request in the example.

The example also utilises **zerolog** for prettier logging.
