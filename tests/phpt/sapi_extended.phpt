--TEST--
extended sapi cookies headers files server
--GET--
page=1
--POST--
action=save
--COOKIE--
sid=abc; theme=dark
--HEADERS--
Host: example.test
User-Agent: php-src-go
Content-Type: text/plain
--FILE--
<?php
echo $_COOKIE['sid'];
echo ':';
echo $_COOKIE['theme'];
echo ':';
echo $_REQUEST['sid'];
echo ':';
echo $_SERVER['REQUEST_METHOD'];
echo ':';
echo $_SERVER['REQUEST_URI'];
echo ':';
echo $_SERVER['HTTP_HOST'];
echo ':';
echo $_SERVER['HTTP_USER_AGENT'];
echo ':';
echo $_SERVER['CONTENT_TYPE'];
echo ':';
echo $_SERVER['HTTP_COOKIE'];
echo ':';
echo count($_FILES);
--EXPECT--
abc:dark:abc:POST:/?page=1:example.test:php-src-go:text/plain:sid=abc; theme=dark:0
