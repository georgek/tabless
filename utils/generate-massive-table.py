#!/usr/bin/env python

import time
import random
import argparse
from signal import signal, SIGPIPE, SIG_DFL
signal(SIGPIPE, SIG_DFL)


LOREM = """Nullam eu ante vel est convallis dignissim Fusce suscipit wisi nec
facilisis facilisis est dui fermentum leo quis tempor ligula erat quis odio
Nunc porta vulputate tellus Nunc rutrum turpis sed pede Sed bibendum Aliquam
posuere Nunc aliquet augue nec adipiscing interdum lacus tellus malesuada
massa quis varius mi purus non odio Pellentesque condimentum magna ut
suscipit hendrerit ipsum augue ornare nulla non luctus diam neque sit amet
urna Curabitur vulputate vestibulum lorem Fusce sagittis libero non molestie
mollis magna orci ultrices dolor at vulputate neque nulla lacinia eros Sed
id ligula quis est convallis tempor Curabitur lacinia pulvinar nibh Nam a
sapien""".split()

MAXINT = 10_000
DEFAULT_NUMROWS = 1_000_000_000
DEFAULT_NUMCOLS = 10
DEFAULT_SLEEP_TIME = 0
DEFAULT_DELIMITER = "\t"


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("-r", "--num_rows", type=int,
                        default=DEFAULT_NUMROWS)
    parser.add_argument("-c", "--num_cols", type=int,
                        default=DEFAULT_NUMCOLS)
    parser.add_argument("-s", "--sleep_time", type=float,
                        default=DEFAULT_SLEEP_TIME)
    parser.add_argument("-d", "--delimiter", type=str,
                        default=DEFAULT_DELIMITER)
    return parser.parse_args()


def random_string(num_words=2, source=LOREM):
    pos = random.randint(0, len(source) - num_words - 1)
    return " ".join(source[pos:pos+num_words])


def main(num_cols=DEFAULT_NUMCOLS,
         num_rows=DEFAULT_NUMROWS,
         sleep_time=DEFAULT_SLEEP_TIME,
         delimiter=DEFAULT_DELIMITER):

    types = [int, float, str]
    col_types = [random.choice(types) for _ in range(num_cols)]

    for i in range(num_rows):
        cells = [str(i)]
        for col_type in col_types:
            if col_type is int:
                cells.append(str(random.randint(0, MAXINT)))
            elif col_type is float:
                cells.append(f"{random.random():.5f}")
            elif col_type is str:
                cells.append(random_string())

        row = delimiter.join(cells)
        print(row, flush=True)
        time.sleep(sleep_time)


if __name__ == '__main__':
    args = parse_args()
    main(**vars(args))
