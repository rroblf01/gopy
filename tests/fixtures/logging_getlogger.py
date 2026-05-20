import logging
import sys


def main() -> None:
    log = logging.getLogger("app")
    log.info("hello")
    log.warning("careful")
    log.error("boom")
    print("done")


if __name__ == "__main__":
    main()
