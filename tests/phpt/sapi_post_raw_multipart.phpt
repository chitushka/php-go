--TEST--
sapi post raw multipart upload
--POST_RAW--
Content-Type: multipart/form-data; boundary=php-src-go
--php-src-go
Content-Disposition: form-data; name="title"

hello
--php-src-go
Content-Disposition: form-data; name="avatar"; filename="hello.txt"
Content-Type: text/plain

abcde
--php-src-go--
--FILE--
<?php
echo $_SERVER['REQUEST_METHOD'];
echo ':';
echo $_SERVER['CONTENT_TYPE'];
echo ':';
echo $_POST['title'];
echo ':';
echo $_REQUEST['title'];
echo ':';
echo $_FILES['avatar']['name'];
echo ':';
echo $_FILES['avatar']['type'];
echo ':';
echo is_file($_FILES['avatar']['tmp_name']);
echo ':';
echo $_FILES['avatar']['error'];
echo ':';
echo $_FILES['avatar']['size'];
--EXPECT--
POST:multipart/form-data; boundary=php-src-go:hello:hello:hello.txt:text/plain:1:0:5
