#!/usr/bin/env python

"""
This is an example script to verify the TRFrecord files generated with terf can
be read using python tensorflow. Each image is displayed using matplotlib.
"""

import sys
import logging
import argparse
import tensorflow as tf
import matplotlib.pyplot as plt

def parse(path, compress=False):
    model = tf.global_variables_initializer()

    with tf.Session() as session:
        session.run(model)

        options = None
        if compress:
            options = tf.python_io.TFRecordOptions(compression_type=tf.python_io.TFRecordCompressionType.ZLIB)

        record_iterator = tf.python_io.tf_record_iterator(path=path, options=options)

        for string_record in record_iterator:
            
            example = tf.train.Example()
            example.ParseFromString(string_record)

            height = int(example.features.feature['image/height']
                                         .int64_list
                                         .value[0])

            width = int(example.features.feature['image/width']
                                        .int64_list
                                        .value[0])

            img_string = (example.features.feature['image/encoded']
                                          .bytes_list
                                          .value[0])

            label = (example.features.feature['image/class/text']
                                        .bytes_list
                                        .value[0])

            fname = (example.features.feature['image/filename']
                                        .bytes_list
                                        .value[0])

            image = tf.image.decode_jpeg(img_string, channels=3)
            image = tf.image.convert_image_dtype(image, dtype=tf.float32)
            plt.imshow(image.eval())
            plt.show()

def main():
    logging.basicConfig(
        format='%(asctime)s [%(levelname)s] %(message)s',
        datefmt='%Y-%m-%d %H:%M:%S',
        level=logging.CRITICAL
    )

    parser = argparse.ArgumentParser(description='Parse a TFRecord file')
    parser.add_argument("-v", "--verbose", help="output debugging information", action="store_true")
    parser.add_argument("-i", "--input", help="Path to the input file")
    parser.add_argument("-z", "--zlib", help="Use zlib compression", action="store_true")

    args = parser.parse_args()
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)


    if not args.input:
        logging.critical("Please specify an input file")
        sys.exit(1)

    parse(args.input, args.zlib)
	

if __name__ == "__main__":
    main()
