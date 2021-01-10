# gfcptun

gfcptun: An fast and low-latency connection tunnel using GFCP over UDP.

## Recommendations:

1. 65535 available files per process, or more.
2. MTU of 9702 is recommended for high-speed local links.
3. Suggested `sysctl` tuning parameters UDP handling:
```
net.core.rmem_max=26214400		// BDP (Bandwidth Delay Product)
net.core.rmem_default=26214400
net.core.wmem_max=26214400
net.core.wmem_default=26214400
net.core.netdev_max_backlog=2048	// (Proportional To Receive Window)
```
4. Increase buffering for high-speed local links to 16MB or more, example:
```
-sockbuf 16777217
```

## Invocation examples:

```
Client: ./gfcp_client -r "LISTEN_IP:4321" -l ":8765" -mode fast3 -nocomp -autoexpire 900 -sockbuf 33554434 -dscp 46
Server: ./gfcp_server -t "TARGET_IP:8765" -l ":4321" -mode fast3 -nocomp -sockbuf 33554434 -dscp 46
```
> Application -> **GFCP Tunnel[8765/TCP] -> GFCP Server(4321/UDP)** -> Server[8765/TCP]

- Other useful paramters: `-mode fast3 -ds 10 -ps 3` etc.

## Tuning for increased throughput:

- To tune, increase `-rcvwnd` on client and `-sndwnd` on server in unison.
  - The mininum window will dictates the maximum link throughput: `( 'Wnd' * ( 'MTU' / 'RTT' ) )`
  - MTU should be set by -mtu paramter and not exceed the MTU of the physical interface.

## Tuning for reduced latency:

- Retransmission algorith aggressiveness:
  - *`fast3` > `fast2` > `fast` > `normal` > `defaulat`*

### Head of line blocking due to N->1 multiplexing:

- Raise `-smuxbuf` 16MB or more - actual value to use depends on link congestion and available memory.
- SMUXv2 can limit per-stream memory usage. Enable with `-smuxver 2`, and tune `-streambuf`.
  - Example: `-smuxver 2 -streambuf 8388608` for 8MiB buffer per stream.
- Start tuning by limting the stream buffer on the **receiving** side of the link.
  - Back-pressure should trigger exustingg congestion control mechanisms, providing practical rate limiting to prevent the exhaustion
    of upstream capacity and downlink starvation (buffer-bloat scenario). 
- SMUXv2 configuration is *not negotiated*, os must be set manually on both sides of the GFCP tunnel.

### Memory Control

- GOGC tuning of 20 for low-memory devices, 100 or higher for servers.

- Notes for SMUX tuning (as per KCP):

  - Primary memory allocation is done from a buffer pool *xmit.Buf*, in the GFCP layer. When allocated a *fixed-capacity* (usually 1500 bytes,
determined by the MtuLimit, will be returned: the *rx queue*, *tx queue* and *fec queue* all allocate from there, and return the bytes to the pool after use.

- The buffer pool mechanism maintains a *high watermark* for *in-flight* objects from the pool, as to survive perodic garbage collection. 

- Memory will be returned to the system by the runtime when idle, as determined by `-sndwnd`,`-rcvwnd`,`-ds`, `-ps` settings. These tunables
affect the *high watermark*: the larger the value, the higher the total memory consumption.

- The `-smuxbuf` setting and GOMAXPROCS adjust the balance between *concurrency limits* and *resource usage*. 
  - Increase smuxbuf will increase practical concurrency limits, however, the `-smuxbuf` value alone is not linerally proprotional server concurrency handling - due to garbage collection interaction, only empiracal testing can provide practical tuning guidelines.

### Compression

- Optional compression using Snappy is available. 

- Compression may save bandwidth for high-compressable data, but will increase overhead in other cases.
  - Compression is enabled by default: use `-nocomp` to disable.  Both ends must use the same setting.

### Monitoring

- Upon receiving the `USR1` signal, detailed link information will be output.

### Low-level GFCP tuning: 

- Example: `-mode manual -nodelay 1 -interval 20 -resend 2 -nc 1`

