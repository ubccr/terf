===============================================================================
terf - TensorFlow TFRecords file format Reader/Writer
===============================================================================

|godoc|

terf is a Go library for reading/writing TensorFlow `TFRecords files
<https://www.tensorflow.org/versions/r1.1/api_guides/python/python_io#tfrecords_format_details>`_.
The goals of this project are two fold:

1. Read/Write TensorFlow TFRecords files in Go
2. Provide an easy way to generate example image datasets for use in TensorFlow

With terf you can easily build, inspect, and extract image datasets from the
command line without having to install TensorFlow. terf was developed for use
with `MARCO <https://marco.ccr.buffalo.edu>`_ but should work with most image
datasets. The TFRecords file format is based on the imagenet dataset from the
inception research model in TensorFlow.

-------------------------------------------------------------------------------
Install
-------------------------------------------------------------------------------

Binaries for your platform can be found `here <https://github.com/ubccr/terf/releases>`_

Usage::

    $ ./terf --help

-------------------------------------------------------------------------------
Examples
-------------------------------------------------------------------------------

~~~~~~~~~~~~~~~~~~~~~~~~~
Create an image dataset
~~~~~~~~~~~~~~~~~~~~~~~~~

You have a directory of images that have been labeled and you want to build an
image dataset that can be used in TensorFlow. First step is to generate a CSV
file in the following format::

	image_path,image_id,label_id,label_text,label_raw,source

Where image_path is the path to the raw image file, image_id is the unique
identifier for an image, label_id is the integer identifier of the normalized
label, label_raw is the integer identifier for the raw label, label_text is the
normalized label, and source is the source (organization/creator etc) that
produced the image. For example::

	image_path,image_id,label_id,label_text,label_raw,source
	/data/03c3_G6_ImagerDefaults_6.jpg,123,1,Crystals,12,101
	/data/X0000056450155200509052032.png,124,0,Clear,15,104


To build the image dataset run the following command::

	$ ./terf -d build --input images.csv --output train_directory/ --size 1024	

This will convert the image data into a sharded data set of TFRecords files in
the train/ output directory::
	
	train_directory/train-00000-of-00024
	train_directory/train-00001-of-00024
	...
	train_directory/train-00023-of-00024

Each TFRecord file will contain ~1024 records. Each record within the TFRecord
file is a serialized Example proto. The Example proto contains the following
fields::

	image/height: integer, image height in pixels
	image/width: integer, image width in pixels
	image/colorspace: string, specifying the colorspace, always 'RGB'
	image/channels: integer, specifying the number of channels, always 3
	image/class/label: integer, specifying the index in a normalized classification layer
	image/class/raw: integer, specifying the index in the raw (original) classification layer
	image/class/source: integer, specifying the index of the source (creator of the image)
	image/class/text: string, specifying the human-readable version of the normalized label
	image/format: string, specifying the format, always 'JPEG'
	image/filename: string containing the basename of the image file
	image/id: integer, specifying the unique id for the image
	image/encoded: string, containing JPEG encoded image in RGB colorspace

~~~~~~~~~~~~~~~~~~~~~~~~~
Inspect an image dataset
~~~~~~~~~~~~~~~~~~~~~~~~~

Generate summary statistics on an image dataset::

	$ ./terf -d summary --input train_directory/
	INFO[0000] Processing file  path=train_directory/train-00001-of-00001 zlib=false
	Total: 10
	Label: 
		- Clear: 5
		- Precipitate: 4
		- Crystals: 1
	Source: 
		- 2: 2
		- 3: 6
		- 1: 2
	Label ID: 
		- 1: 1
		- 0: 5
		- 3: 4
	Label Raw: 
		- 30: 1
		- 2: 3
		- 8: 1
		- 16: 1
		- 1: 2
		- 14: 2

~~~~~~~~~~~~~~~~~~~~~~~~~
Extract an image dataset
~~~~~~~~~~~~~~~~~~~~~~~~~

Extract the raw image data from a dataset::

	$ ./terf -d extract --input train_directory -o dump/
	INFO[0000] Processing file    path=train_directory/train-00001-of-00001 zlib=false
	$ find dump/
	dump/
	dump/info.csv
	dump/Clear
	dump/Clear/396612.jpg
	dump/Clear/90089.jpg
	dump/Clear/192089.jpg
	dump/Clear/283709.jpg
	dump/Clear/82162.jpg
	dump/Precipitate
	dump/Precipitate/286612.jpg
	dump/Precipitate/421709.jpg
	dump/Precipitate/296118.jpg
	dump/Precipitate/163507.jpg
	dump/Crystals
	dump/Crystals/80373.jpg


~~~~~~~~~~~~~~~~~~~~~~
Go
~~~~~~~~~~~~~~~~~~~~~~

Parse TFRecords file in Go:

.. code-block:: go

	// Open TFRecord file
	in, err := os.Open("train-000")
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	r := terf.NewReader(in)

	count := 0
	for {
		// example will be a TensorFlow Example proto
		example, err := r.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		// Do something with example

		id := terf.ExampleFeatureInt64(example, "image/id")
		labelID := terf.ExampleFeatureInt64(example, "image/class/label")
		labelText := string(terf.ExampleFeatureBytes(example, "image/class/text"))

		fmt.Printf("Image: %d Label: %s (%d)\n", id, labelText, labelID)
		count++
	}

	fmt.Printf("Total records: %d\n", count)

-------------------------------------------------------------------------------
License
-------------------------------------------------------------------------------

terf is released under the GPLv3 License. See the LICENSE file.

.. |godoc| image:: https://godoc.org/github.com/golang/gddo?status.svg
    :target: https://godoc.org/github.com/ubccr/terf
    :alt: Godoc
