# gfcptun

gfcptun: An fast and low-latency connection tunnel using GFCP over UDP.

----

## Basic gfcptun/GFCP recommendations

- Make available 65535 or more file descriptors per gfcptun process.

- MTU of 9000-9702 is recommended for high-speed local links.

- Suggested minimum `sysctl` tuning parameters for Linux UDP handling:

  - (See
    <https://www.sciencedirect.com/topics/computer-science/bandwidth-delay-product>
    for additional BDP background information.)

```shell
net.core.rmem_max=26214400       # Tune for BDP (bandwidth delay product)
net.core.rmem_default=26214400
net.core.wmem_max=26214400
net.core.wmem_default=26214400
net.core.netdev_max_backlog=2048 # (Adjust proportional to receive window)
```

- Increase buffering for high-speed local links to 16MiB or more, example:

```text
-sockbuf 16777217
```

----

## Process invocation examples

```shell
client -r "IN:4321" -l ":8765" -mode fast3 -nocomp -autoexpire 900 -sockbuf 33554434 -dscp 46
server -t "OUT:8765" -l ":4321" -mode fast3 -nocomp -sockbuf 33554434 -dscp 46
```

- Application 🠚 Out (8765/TCP) 🠚 Internet 🠚 In (4321/UDP) 🠚 Server (8765/TCP)

  - Other useful parameters: `-mode fast3 -ds 10 -ps 3`, etc.

----

## Tuning for increased total throughput

- To tune, increase `-rcvwnd` on client, and `-sndwnd` on server, in unison.

  - The minimum window size will dictate the maximum link throughput:
    `( 'Wnd' * ( 'MTU' / 'RTT' ) )`

  - MTU should be set by -mtu parameter and never exceed the MTU of the physical
    interface. For DC/high-speed local links w/jumbo framing, using an MTU of
    9000-9702 is highly recommended.

----

## Tuning for reduced overall latency

- Adjust the retransmission algorithm aggressiveness:

  - _`fast3` *🠚* `fast2` *🠚* `fast` *🠚* `normal` *🠚* `default`_

----

## Avoiding **N** _🠚_ **1** multiplexing [head-of-line blocking](https://www.sciencedirect.com/topics/computer-science/head-of-line-blocking) behavior

- Raise `-smuxbuf` to 16MiB (or more), however, the actual value to use depends
  on link congestion as well as available contiguous system memory.

- SMUXv2 can be used to limit per-stream memory usage. Enable with `-smuxver 2`,
  and then tune with `-streambuf` _(size in bytes)_.

  - Example: `-smuxver 2 -streambuf 8388608` for an 8MiB buffer (per stream).

- Start tuning by limiting the stream buffer on the **receiving** side of the
  link.

  - Back-pressure should trigger existing congestion control mechanisms,
    providing practical rate limiting to prevent the exhaustion of upstream
    capacity and also avoiding downlink starvation (bufferbloat scenario).

- SMUXv2 configuration is _not negotiated_, so must be set manually on **both**
  sides of the GFCP link.

----

## Memory consumption control

- `GOGC` runtime environment variable tuning recommendation:

  - **10**-**20** for low-memory systems and embedded devices

  - **120**-**150** (or higher) for dedicated servers

- Notes regarding (GF)SMUX(v1/v2) tuning:

  - Primary memory allocation is done from a buffer pool (_xmit.Buf_), in the
    GFCP layer. When allocating, a _fixed-size_ buffer, determined by the
    MtuLimit, will be returned. From there, the _rx queue_, _tx queue_, and,
    _fec queue_ will be allocated, and will return the allocation to the buffer
    pool after use.

- The buffer pool mechanism maintains a _high watermark_ for _in-flight_ objects
  from the pool to survive periodic runtime garbage collection.

- Memory will be returned to the system by the Go runtime when idle. Variables
  that can be used for tuning this are `-sndwnd`,`-rcvwnd`,`-ds`, and `-ps`.

  - These parameters affect the _high watermark_ - the larger the value, the
    higher the total memory consumption can be at any given moment.

- The `-smuxbuf` setting and `GOMAXPROCS` variable can be used to tune the
  balance between the _concurrency limit_ and overall _resource usage_.

  - Increasing `-smuxbuf` will increase the practical concurrency limit,
    however, the `-smuxbuf` value is **not** linerally proprotional to the
    concurrency handling maximum because Go runtime's garbage collection is, for
    practical purposes, non-deterministic.

  - Only empirical testing can provide the feedback required for real-world link
    tuning and optimization.

----

## Link compression

- Optional compression (using _Snappy_) is supported.

- Compression saves bandwidth on _redundant, low-entropy_ data, but **will**
  **increase** overhead (and CPU usage) in **all** other cases.

  - Compression is **enabled by default**: use `-nocomp` to disable.

    - Both ends of the link **must** use the same compression setting.

----

## GFCP SNSI monitoring

- Upon receiving a `USR1` signal, detailed link information will be displayed.

----

## Low-level GFCP tuning

- Example: `-mode manual -nodelay 1 -interval 20 -resend 2 -nc 1`

----

## Availability

- [GitHub](https://github.com/johnsonjh/gfcptun)
- [GitLab](https://gitlab.com/johnsonjh/gfcptun)
- [SourceHut](https://sr.ht/~trn/gfcptun)
- [NotABug](https://notabug.org/trn/gfcptun)

----
