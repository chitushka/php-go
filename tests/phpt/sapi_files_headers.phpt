--TEST--
sapi files and response headers
--FILES--
avatar.name=hello.txt
avatar.type=text/plain
avatar.tmp_name=/tmp/php123
avatar.error=0
avatar.size=5
--FILE--
<?php
echo $_FILES['avatar']['name'];
echo ':';
echo $_FILES['avatar']['type'];
echo ':';
echo $_FILES['avatar']['tmp_name'];
echo ':';
echo $_FILES['avatar']['error'];
echo ':';
echo $_FILES['avatar']['size'];
echo '|';
header('X-Test: one');
header('X-Test: two');
header('X-Trace: a', false);
foreach (headers_list() as $header) {
    echo $header;
    echo '|';
}
--EXPECT--
hello.txt:text/plain:/tmp/php123:0:5|X-Test: two|X-Trace: a|
