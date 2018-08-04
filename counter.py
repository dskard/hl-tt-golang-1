#!/usr/bin/env python

import argparse
import logging
import signal
import sys
import time

logger = None

def parseoptions():
    usage = "usage: %prog [options]"
    parser = argparse.ArgumentParser(usage=usage)

    parser.add_argument('--start',
                    help='Where to start counting from.',
                    action='store',
                    dest='start',
                    default=0,
                    type=int)

    parser.add_argument('remainder', nargs=argparse.REMAINDER)

    options = parser.parse_args()

    return options,options.remainder


def start_logging(options):

        logger = logging.getLogger(__name__)

        # setup a log file
        logfile = "counter.log"
        loglevel = logging.INFO
        logformat = '%(asctime)s %(message)s'

        logging.basicConfig(
                filename=logfile,
                level=loglevel,
                format=logformat)

        logger.info("starting %s" % sys.argv[0])
        logger.info("command line options: %s" % sys.argv[1:])

        # print out the parsed options
        logger.debug('opts = {}'.format(options))


def start_counting(startval):

    i = startval

    while True:
        print(i)
        i += 1
        time.sleep(1)


if __name__ == "__main__":

    # parse command line options
    options,remainder = parseoptions()

    # start logging
    start_logging(options)

    def signal_handler(sig,dummy):
        sys.exit(1)

    signal.signal(signal.SIGINT, signal_handler)

    start_counting(options.start)

    sys.exit(0)
