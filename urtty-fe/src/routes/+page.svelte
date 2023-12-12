<script>
    import { onMount } from 'svelte';
    import { writable } from 'svelte/store';
    import Urbit from '@urbit/http-api';
    import { Terminal } from 'xterm';
    import '../../node_modules/xterm/css/xterm.css';

    let terminalContainer;
    let term;
    let inputBuffer = "";
    const urbit = new Urbit("");
    export const broadcast = writable("");

    export const subscribe = ship => {
        urbit.ship = ship;
        urbit.onOpen = () => console.log("onOpen opened");
        urbit.onRetry = () => console.log("onRetry called");
        urbit.onError = e => console.error("onError: " + e);
        urbit.subscribe({
            app: "urtty",
            path: "/broadcast",
            event: handleEvent,
            quit: handleQuit,
            err: handleErr
        });
    };

    export const sendPoke = payload => {
        urbit.poke({
        app: "urtty",
        mark: "action",
        json: {"action":payload},
            onSuccess: handlePokeSuccess,
            onError: handlePokeError
        })
    }

    const handlePokeSuccess = () => {
        console.log("poke succeeded")
    }

    const handlePokeError = event => {
        console.log(event)
    }

    const handleEvent = event => {
        if (typeof event.cord === 'string') {
            let broadcast;
            try {
                broadcast = JSON.parse(event.cord);
            } catch (error) {
                console.error("Failed to parse: ", error);
                return;
            }
            handleBroadcast(broadcast);
        }
    };

    const handleQuit = () => console.error("quit called");
    const handleErr = () => console.error("error called");

    const handleBroadcast = broadcast => {
        if (broadcast && broadcast.broadcast) {
            const decodedData = atob(broadcast.broadcast);
            term.write(decodedData);
        }
    };

    const sendDataToUrbit = data => {
        const encodedData = btoa(data);
        const jsonData = JSON.stringify({ action: encodedData });
        sendPoke(data);
    };

    onMount(() => {
        term = new Terminal({cursorBlink: true, convertEol: true});
        term.open(terminalContainer);
        term.writeln('Requesting a shell...');
        subscribe(window.ship);
        sendDataToUrbit("init");
        term.onData(key => {
            if (key === '\r') {
                sendDataToUrbit(inputBuffer + '\r');
                term.write('\r\n');
                inputBuffer = "";
            } else if (key === '\x7F' || key === '\b') {
                if (inputBuffer.length > 0) {
                    inputBuffer = inputBuffer.slice(0, -1);
                    term.write('\b \b');
                }
            } else {
                inputBuffer += key;
                term.write(key);
            }
        });
    });
</script>

<main>
    <body>
        <div class="title">
            <h2>$> urtty</h2>
            <h3><img src="urtty/np.png"> by native planet</h3>
        </div>
        <div id="terminal-container">
            <div bind:this={terminalContainer} id="terminal-inner"></div>
        </div>
    </body>
</main>

<style>
@import url('https://fonts.googleapis.com/css?family=Space+Mono&display=swap');
body {
    font-family: 'Arial', sans-serif;
    background-color: #4a4a4a;
    color: #c8c8c8;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;
}

#terminal-container {
    background-color: #000;
    border-radius: 25px;
    padding: 10px;
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.3);
    margin: 20px;
    width: 80%;
    max-width: 800px;
    height: auto;
    max-height: 800px;
    overflow: auto;
    position: relative;
    scrollbar-width: thin;
    scrollbar-color: #4a4a4a #1d1f21;
}

#terminal-inner {
    width: 100%;
    height: auto;
}

.title { 
    color: #fff; 
    font-family: 'Space Mono', monospace; 
    padding-bottom: 10px;
    text-align: center;
}
.title h2 {
    font-size: 24px;
}
.title h3 {
    font-size: 16px;
    line-height: 1;
    display: flex;
    align-items: center;
}

.title h3 img {
    width: auto;
    margin-right: 8px;
}
</style>