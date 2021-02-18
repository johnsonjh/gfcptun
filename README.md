# gfcptun

gfcptun: An fast and low-latency connection tunnel using GFCP over UDP.

[![Codacy Badge](https://api.codacy.com/project/badge/Grade/a01d5d75fe8143e0b1a6962f3e54ae14)](https://app.codacy.com/gh/gridfinity/gfcptun?utm_source=github.com&utm_medium=referral&utm_content=gridfinity/gfcptun&utm_campaign=Badge_Grade)

---

## GFCP Recommendations

- 65535 available files per process, or more.
- MTU of 9702 is recommended for high-speed local links.
- Suggested `sysctl` tuning parameters for UDP handling. (See
  <https://www.sciencedirect.com/topics/computer-science/bandwidth-delay-product>
  for BDP background information):

```shell
net.core.rmem_max=26214400             # BDP (Bandwidth Delay Product)
net.core.rmem_default=26214400
net.core.wmem_max=26214400
net.core.wmem_default=26214400
net.core.netdev_max_backlog=2048       # (Proportional To Receive Window)
```

- Increase buffering for high-speed local links to 16MB or more, example:

```text
-sockbuf 16777217
```

## Invocation examples

```shell
client -r "LISTEN:4321" -l ":8765" -mode fast3 -nocomp -autoexpire 900 -sockbuf 33554434 -dscp 46
server -t "TARGET:8765" -l ":4321" -mode fast3 -nocomp -sockbuf 33554434 -dscp 46
```

> Application🠚TunOut[8765/TCP]🠚Internet🠚TunIn[4321/UDP]🠚Server[8765/TCP]

- Other useful parameters example: `-mode fast3 -ds 10 -ps 3`, etc.

## Tuning for increased total throughput

- To tune, increase `-rcvwnd` on client, and `-sndwnd` on server, in unison.
  - The minimum window size will dictate the maximum link throughput:
    `( 'Wnd' * ( 'MTU' / 'RTT' ) )`
  - MTU should be set by -mtu parameter and never exceed the MTU of the physical
    interface. For DC/high-speed local links w/jumbo framing, using an MTU of
    9000-9702 is highly recommended.

## Tuning for reduced overall latency

- Retransmission algorithm aggressiveness:
  - _`fast3`🠚`fast2`🠚`fast`🠚`normal`🠚`default`_

### Avoiding [Head-of-line blocking](https://www.sciencedirect.com/topics/computer-science/head-of-line-blocking) due to N🠚1 multiplexing

- Raise `-smuxbuf` to 16MB (or more) - the actual value to use depends on
  average link congestion and available system memory.
- SMUXv2 can be used to limit per-stream memory usage. Enable with `-smuxver 2`,
  and then tune with `-streambuf` (size in bytes).

  - Example: `-smuxver 2 -streambuf 8388608` for an 8MiB buffer (per stream).

- Start tuning by limiting the stream buffer on the **receiving** side of the
  link.

  - Back-pressure should trigger existing congestion control mechanisms,
    providing practical rate limiting to prevent the exhaustion of upstream
    capacity and also avoiding downlink starvation (bufferbloat scenario).

- SMUXv2 configuration is _not negotiated_. It must be set manually on **both**
  sides of the GFCP link.

### Memory Control

- `GOGC` runtime environment variable tuning recommendations:

  - **20** for low-memory devices
  - **120** (or higher) for servers

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
  balance between _concurrency limits_ and overall _resource usage_.

  - Increasing `-smuxbuf` will increase the practical concurrency limit,
    however, the `-smuxbuf` value is **not** linerally proprotional to the total
    concurrency handling maximum, due to the Go runtime's non-deterministic
    garbage collection. Because of this, only empirical testing can provide the
    data needed for real-life tuning recommendations.

### Link compression

- Optional compression using Snappy is available.

- Compression may save bandwidth on _redundant, low-entropy_ data, but **will**
  **increase** overhead in all other cases (and increase CPU usage).

  - Compression is **enabled by default**: use `-nocomp` to disable.

    - Both ends of the link **must** use the same compression setting.

### GFCP SNSI monitoring

- Upon receiving a `USR1` signal, detailed link information will be displayed.

### Low-level GFCP tuning

- Example: `-mode manual -nodelay 1 -interval 20 -resend 2 -nc 1`
