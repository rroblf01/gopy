import signal
import atexit
import gc


def on_exit() -> None:
    pass


def main() -> None:
    print(signal.SIGINT == 2)
    print(signal.SIGTERM == 15)
    atexit.register(on_exit)
    print(gc.isenabled())
    n: int = gc.collect()
    print(n >= 0)


if __name__ == "__main__":
    main()
