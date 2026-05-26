import fcntl
import os


def main() -> None:
    pair = os.pipe()
    fd = pair[0]
    fcntl.flock(fd, fcntl.LOCK_SH | fcntl.LOCK_NB)
    fcntl.flock(fd, fcntl.LOCK_UN)
    os.close(fd)
    os.close(pair[1])
    print("ok")


if __name__ == "__main__":
    main()
