--TEST--
sapi uploaded file move and array uploads
--POST_RAW--
Content-Type: multipart/form-data; boundary=php-src-go-upload
--php-src-go-upload
Content-Disposition: form-data; name="avatar"; filename="hello.txt"
Content-Type: text/plain

hello
--php-src-go-upload
Content-Disposition: form-data; name="docs[]"; filename="a.txt"
Content-Type: text/plain

a
--php-src-go-upload
Content-Disposition: form-data; name="docs[]"; filename="b.txt"
Content-Type: text/plain

bb
--php-src-go-upload--
--FILE--
<?php
$tmp = $_FILES['avatar']['tmp_name'];
echo is_uploaded_file($tmp);
echo ':';
echo file_get_contents($tmp);
echo ':';
echo move_uploaded_file($tmp, 'tests/phpt/upload_moved.tmp');
echo ':';
echo is_uploaded_file($tmp);
echo ':';
echo file_get_contents('tests/phpt/upload_moved.tmp');
echo ':';
echo $_FILES['docs']['name'][0];
echo ':';
echo $_FILES['docs']['size'][0];
echo ':';
echo $_FILES['docs']['name'][1];
echo ':';
echo $_FILES['docs']['size'][1];
--CLEAN--
<?php
unlink('tests/phpt/upload_moved.tmp');
--EXPECT--
1:hello:1::hello:a.txt:1:b.txt:2
