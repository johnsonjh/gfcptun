# gfcptun

gfcptun: An fast and low-latency connection tunnel using GFCP over UDP.

[![Codacy Badge](https://api.codacy.com/project/badge/Grade/a01d5d75fe8143e0b1a6962f3e54ae14)](https://app.codacy.com/gh/gridfinity/gfcptun?utm_source=github.com&utm_medium=referral&utm_content=gridfinity/gfcptun&utm_campaign=Badge_Grade)

---

## GFCP Recommendations

1. 65535 available files per process, or more.
2. MTU of 9702 is recommended for high-speed local links.
3. Suggested `sysctl` tuning parameters UDP handling - see <https://www.sciencedirect.com/topics/computer-science/bandwidth-delay-product> for BDP background information:

```text
net.core.rmem_max=26214400  // BDP (Bandwidth Delay Product)
net.core.rmem_default=26214400
net.core.wmem_max=26214400
net.core.wmem_default=26214400
net.core.netdev_max_backlog=2048 // (Proportional To Receive Window)
```

1. Increase buffering for high-speed local links to 16MB or more, example:

```text
-sockbuf 16777217
```

## Invocation examples

```text
Client: ./gfcp_client -r "LISTEN_IP:4321" -l ":8765" -mode fast3 -nocomp -autoexpire 900 -sockbuf 33554434 -dscp 46
Server: ./gfcp_server -t "TARGET_IP:8765" -l ":4321" -mode fast3 -nocomp -sockbuf 33554434 -dscp 46
```

> Application🠚TunOut[8765/TCP]🠚Internet🠚TunIn[4321/UDP]🠚Server[8765/TCP]

- Other useful parameters example: `-mode fast3 -ds 10 -ps 3` etc.

## Tuning for increased total throughput

- To tune, increase `-rcvwnd` on client and `-sndwnd` on server in unison.
  - The minimum window will dictates the maximum link throughput:
    `( 'Wnd' * ( 'MTU' / 'RTT' ) )`
  - MTU should be set by -mtu parameter and not exceed the MTU of the physical
    interface. For DC local links w/jumbo framing, MTU of 9000+ recommended.

## Tuning for reduced overall latency

- Retransmission algorithm aggressiveness:
  - _`fast3`🠚`fast2`🠚`fast`🠚`normal`🠚`default`_

### [Head-of-line blocking](https://www.sciencedirect.com/topics/computer-science/head-of-line-blocking) due to N🠚1 multiplexing

- Raise `-smuxbuf` 16MB or more - actual value to use depends on link congestion
  and available memory.
- SMUXv2 can limit per-stream memory usage. Enable with `-smuxver 2`, and tune
  `-streambuf`.

  - Example: `-smuxver 2 -streambuf 8388608` for 8MiB buffer per stream.

- Start tuning by limiting the stream buffer on the **receiving** side of the
  link.

  - Back-pressure should trigger existing congestion control mechanisms,
    providing practical rate limiting to prevent the exhaustion of upstream
    capacity and downlink starvation (bufferbloat scenario).

- SMUXv2 configuration is _not negotiated_, so must be set manually on **both**
  sides of the GFCP link.

### Memory Control

- `GOGC` varuable tuning:

  - **20** for low-memory devices
  - **120** (or higher) for servers

- Notes for SMUX tuning:

  - Primary memory allocation is done from a buffer pool (_xmit.Buf_), in the
    GFCP layer. When allocated, a _fixed-size_ allocation determined by the
    MtuLimit, will be returned. From there, the _rx queue_, _tx queue_, and,
    _fec queue_ are allocated, returning the allocation to the pool after use.

- The buffer pool mechanism maintains a _high watermark_ for _in-flight_ objects
  from the pool, as to survive periodic garbage collection.

- Memory will be returned to the system by the Go runtime when idle. Variables
  that can be used for tuning this are `-sndwnd`,`-rcvwnd`,`-ds`, and `-ps`.
  These parameters affect the _high watermark_ - the larger the value, the
  higher the total memory consumption at any given moment.

- The `-smuxbuf` setting and `GOMAXPROCS` variable adjust the balance between
  _concurrency limits_ and overall _resource usage_.
  - Increasing `-smuxbuf` will increase practical concurrency limits, however,
    the `-smuxbuf` value is **not** linerally proprotional to total concurrency
    handling, mostly due to the runtime's non-deterministic garbage collection
    interactions. Because of this, only empirical testing can provide feedback
    needed for making usable, real-life tuning recommendations.

### Compression

- Optional compression using Snappy is available.

- Compression may save bandwidth for _redundant, low-entropy_ data, but **will**
  **increase** overhead in all other cases (and increase CPU usage).
  - Compression is **enabled by default**: use `-nocomp` to disable.
    - Both ends of the link **must** use the same compression setting, as it is
      not negotiated.

### Monitoring

- Upon receiving a `USR1` signal, detailed link information will be displayed.

### Low-level GFCP tuning

- Example: `-mode manual -nodelay 1 -interval 20 -resend 2 -nc 1`
