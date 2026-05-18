def describe(**fields: int) -> int:
    total: int = 0
    for k in fields:
        total += int(fields[k])
    return total


def main() -> None:
    print(describe(a=1, b=2, c=3))
    payload: dict[str, int] = {"x": 10, "y": 20}
    # **payload splices the dict entries into the kwargs slot.
    print(describe(**payload))
    # Mixed: explicit kwargs + splat both flow into fields.
    print(describe(z=100, **payload))


if __name__ == "__main__":
    main()
