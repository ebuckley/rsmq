const RedisSMQ = require("rsmq");

const rsmq = new RedisSMQ( {host: "127.0.0.1", port: 6379, ns: "rsmq"} );

const QUEUE = "test";

rsmq.createQueue({ qname: QUEUE }, function (err, resp) {
    if (err) {
        console.error(err)
        return
    }

    if (resp === 1) {
        console.log("queue created")
    }
});

rsmq.getQueueAttributes({ qname: QUEUE }, function (err, resp) {
    if (err) {
        console.error(err);
        return;
    }

    console.log("==============================================");
    console.log("=================Queue Stats==================");
    console.log("==============================================");
    console.log("visibility timeout: ", resp.vt);
    console.log("delay for new messages: ", resp.delay);
    console.log("max size in bytes: ", resp.maxsize);
    console.log("total received messages: ", resp.totalrecv);
    console.log("total sent messages: ", resp.totalsent);
    console.log("created: ", resp.created);
    console.log("last modified: ", resp.modified);
    console.log("current n of messages: ", resp.msgs);
    console.log("hidden messages: ", resp.hiddenmsgs);
});

rsmq.sendMessage({ qname: QUEUE, message: "Hello World "}, function (err, resp) {
    if (err) {
        // prob queue already exists...
        console.log(typeof err);
        return
    }
    shout('Send Message');

    console.log(resp);
});

rsmq.receiveMessage({ qname: QUEUE }, function (err, resp) {
    if (err) {
        console.error(err)
        return
    }
    shout('RecieveMessage');
    if (resp.id) {
        console.log("Message received.", resp)
    } else {
        console.log("No messages for me...")
    }
});



// utility

const shout = (msgContent) => {
    const bar = '==============================================';
    let padding = (bar.length / 2) - (msgContent.length / 2);
    let trimTail = false;
    if (Math.floor(padding) !== padding) {
        padding = Math.floor(padding);
        trimTail = true;
    }
    const padBar = Array.apply(null, Array(padding - 1))
        .map(function () {}).reduce((acc, val) => acc + '=', '=');
    console.log(bar);
    console.log(padBar + msgContent  + (trimTail ? padBar.slice(-1) : padBar));
    console.log(bar);
}