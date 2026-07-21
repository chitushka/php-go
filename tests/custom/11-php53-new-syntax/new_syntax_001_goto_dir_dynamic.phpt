--TEST--
New syntax 001 (NEW in PHP 5.3): goto statement, __DIR__ constant, dynamic static property/method access
--FILE--
<?php
// goto - NEW in PHP 5.3
$i = 0;
start:
$i++;
if ($i < 5) {
    goto start;
}
var_dump($i);

$found = false;
foreach (array(1, 2, 3, 4, 5) as $v) {
    if ($v === 3) {
        $found = true;
        goto done;
    }
}
done:
var_dump($found);

// __DIR__ - NEW in PHP 5.3 (magic constant)
var_dump(__DIR__ === dirname(__FILE__));
var_dump(is_string(__DIR__));

// Dynamic static property/method access
class Config {
    public static $env = 'production';
    public static function get($key) {
        return self::${$key};
    }
}
$prop = 'env';
var_dump(Config::$$prop);
var_dump(Config::get('env'));

$className = 'Config';
var_dump($className::$env);

class Factory {
    public static function build() { return 'built'; }
}
$method = 'build';
var_dump(Factory::$method());

$obj = new Factory();
var_dump($obj::build());
--EXPECT--
int(5)
bool(true)
bool(true)
bool(true)
string(10) "production"
string(10) "production"
string(10) "production"
string(5) "built"
string(5) "built"
