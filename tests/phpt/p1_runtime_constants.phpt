--TEST--
P1 bare constants and define-style runtime constants
--FILE--
<?php
define('APP_NAME', 'php-go');
define('ANSWER', 42);
define('MiXeD_NAME', 'case', true);

class RuntimeConstBox {
    const LABEL = 'class-const';
}

echo APP_NAME;
echo "\n";
echo ANSWER + 1;
echo "\n";
echo defined('APP_NAME') ? 'yes' : 'no';
echo "\n";
echo constant('APP_NAME');
echo "\n";
echo defined('TRUE') ? 'builtin' : 'missing';
echo "\n";
echo constant('FALSE') ? 'bad' : 'false';
echo "\n";
echo mixed_name;
echo "\n";
echo defined('RuntimeConstBox::LABEL') ? constant('RuntimeConstBox::LABEL') : 'missing';
echo "\n";
echo MISSING_CONST;
?>
--EXPECT--
php-go
43
yes
php-go
builtin
false
case
class-const
MISSING_CONST
