import errno
import signal


def main() -> None:
    print(errno.EAGAIN)
    print(errno.EWOULDBLOCK)
    print(errno.ECONNRESET)
    print(errno.EADDRINUSE)
    print(errno.EHOSTUNREACH)
    print(errno.ENOTEMPTY)
    print(signal.SIGCHLD)
    print(signal.SIGCONT)
    print(signal.SIGSEGV)
    print(signal.SIGABRT)


if __name__ == "__main__":
    main()
