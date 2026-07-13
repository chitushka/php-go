--TEST--
sapi get post env sections
--GET--
name=php&version=5.3
--POST--
action=run
--ENV--
APP_ENV=test
--FILE--
<?php
echo $_GET['name'];
echo ':';
echo $_GET['version'];
echo ':';
echo $_POST['action'];
echo ':';
echo $_REQUEST['name'];
echo ':';
echo $_REQUEST['action'];
echo ':';
echo $_ENV['APP_ENV'];
echo ':';
echo $_SERVER['REQUEST_METHOD'];
echo ':';
foreach ($_GET as $k => $v) {
    echo $k;
    echo $v;
}
--EXPECT--
php:5.3:run:php:run:test:POST:namephpversion5.3
