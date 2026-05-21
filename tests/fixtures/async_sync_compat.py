import asyncio


async def fetch(x: int) -> int:
    return x * 2


async def compute() -> int:
    a: int = await fetch(3)
    b: int = await fetch(5)
    return a + b


def main() -> None:
    print(asyncio.run(compute()))


if __name__ == "__main__":
    main()
