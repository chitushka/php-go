--TEST--
sapi upload limits and rfc1867 edges
--INI--
upload_max_filesize=3
max_file_uploads=3
--POST_RAW--
Content-Type: multipart/form-data; boundary=php-src-go-limits
--php-src-go-limits
Content-Disposition: form-data; name="MAX_FILE_SIZE"

2
--php-src-go-limits
Content-Disposition: form-data; name="ok"; filename="ok.txt"
Content-Type: text/plain

aa
--php-src-go-limits
Content-Disposition: form-data; name="formbig"; filename="form.txt"
Content-Type: text/plain

aaa
--php-src-go-limits
Content-Disposition: form-data; name="inibig"; filename="ini.txt"
Content-Type: text/plain

aaaa
--php-src-go-limits
Content-Disposition: form-data; name="empty"; filename=""
Content-Type: text/plain

--php-src-go-limits
Content-Disposition: form-data; name="ignored"; filename="ignored.txt"
Content-Type: text/plain

x
--php-src-go-limits--
--FILE--
<?php
echo $_POST['MAX_FILE_SIZE'];
echo ':';
echo $_FILES['ok']['size'];
echo ':';
echo $_FILES['ok']['error'];
echo ':';
echo is_uploaded_file($_FILES['ok']['tmp_name']);
echo ':';
echo $_FILES['formbig']['error'];
echo ':';
echo $_FILES['formbig']['tmp_name'];
echo ':';
echo $_FILES['formbig']['size'];
echo ':';
echo $_FILES['inibig']['error'];
echo ':';
echo $_FILES['inibig']['size'];
echo ':';
echo $_FILES['empty']['error'];
echo ':';
echo $_FILES['empty']['name'];
echo ':';
echo $_FILES['empty']['size'];
echo ':';
echo count($_FILES);
--EXPECT--
2:2:0:1:2::3:1:4:4::0:4
