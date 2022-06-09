# conn-test-go
golang simple code to test upload, download and bidirectional speed comparison for different TCP/UDP protocols

Supports:

1. TCP
3. SCTP, https://github.com/vikulin/sctp
4. QUIC, https://github.com/lucas-clemente/quic-go
5. UDT, https://github.com/vikulin/go-udt, need to do fixes in go-udt.

Used fast xxh3 hashing lib: https://github.com/zeebo/xxh3
