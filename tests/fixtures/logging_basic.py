import logging


def main() -> None:
    logging.basicConfig(level=20)
    logging.info("starting")
    logging.warning("be careful")
    logging.error("boom")
    # logging output goes to stderr; stdout is empty for this fixture.
    print("done")


if __name__ == "__main__":
    main()
