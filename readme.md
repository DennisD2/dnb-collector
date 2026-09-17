## Deutsche Nationalbibliothek OAI API example code
DNB offers open access to some of their document collections. 
These collection can be accessed via some JavaScript-based
viewer frontend. Next image shows the official bookviewer frontend.

![bookviewer-example.png](doc/bookviewer-example.png)

DNB also offer an API, called OAI. Using this API, the documents 
can be accessed using some arbitrary programming language.

## What does this repository contain?
This repository contains code to access one of these collections,
the TGL (Technische Normen, Gütevorschriften und Lieferbedingungen) 
collection from GDR, which was like DIN for BRD.

It is only example code, that can retrieve a single TGL document
artifact tree. 

The documents are named e.g. *TGL 32565*. With the input *32565*, all
related objects from the TGL collection can be downloaded. These
are metadata, OCR data, and TIFF scans of the original documents.
A set, which is like a version of a document, is downloaded in a ZIP
file and contains the mentioned artifacts for this version.

The ZIP file name is the one defined by DNB, and looks like *1253650947X.zip*.

Next image shows content of some of these ZIP files. Per page,
there is a TIFF image. Besides that, there is also OCR information
and some metadata.

![zipcontent.png](doc/zipcontent.png)

If there are multiple versions of a document, all versions are downloaded,
in separate ZIP files. All ZIP files for a document are downloaded to a newly created
directory named like the document. 

Besides ZIP files, there seem to be additional artifacts. I haven't
checked these, because I was only interested in the TIFF scans.
These links seem to be protected. Result of the trial to download
these links are put in files named "download-noname-%d.txt".
Currently, these files contain only error messages.

So, downloading document 32565 results in the following structure.

![doctree.png](doc/doctree.png)

## Disclaimer
I did this code only for fun. I needed a single document from this
collection, and got it using the DNBs own document viewer. But when
I saw that there is an Open API, I could not refrain myself to
try access with Go. 

So don't expect great code quality. It is just a quick hack to
explore the API. Code was partly developed with Googles Gemini AI support.

## OAI API
* API Description (Python based) - https://mybinder.org/v2/gh/deutsche-nationalbibliothek/dnblab/HEAD?filepath=DNB_OAI_Tutorial.ipynb

## Data sets
* Mehr als 18.000 Technische Normen, Gütevorschriften und Lieferbedingungen (TGL) der DDR von 1949 bis 1989. https://www.dnb.de/DE/Professionell/Services/WissenschaftundForschung/DNBLab/DNBLabDatensets/ThematischeSammlungen/technischeNormen.html
* Online browser-driven access to data sets - https://portal.dnb.de/opac/simpleSearch?query=cod%3D2d010&cqlMode=true
