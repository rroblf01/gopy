import logging


def main() -> None:
    log = logging.getLogger("app")

    # Default threshold is WARNING (30) — DEBUG/INFO get filtered.
    print(log.isEnabledFor(logging.DEBUG))
    print(log.isEnabledFor(logging.INFO))
    print(log.isEnabledFor(logging.WARNING))
    print(log.isEnabledFor(logging.ERROR))

    # Drop the threshold so INFO+ passes.
    log.setLevel(logging.INFO)
    print(log.getEffectiveLevel())
    print(log.isEnabledFor(logging.DEBUG))
    print(log.isEnabledFor(logging.INFO))

    # Raise the threshold so only ERROR+ passes.
    log.setLevel(logging.ERROR)
    print(log.isEnabledFor(logging.WARNING))
    print(log.isEnabledFor(logging.ERROR))


if __name__ == "__main__":
    main()
