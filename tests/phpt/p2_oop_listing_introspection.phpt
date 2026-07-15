--TEST--
P2 OOP listing introspection functions
--FILE--
<?php
class ListBase {
    public $base = 'base';
    protected $hidden = 'hidden';
    public static $root = 'root';
    protected static $hiddenRoot = 'hidden-root';
    public function baseMethod() {}
    protected function hiddenMethod() {}
}

class ListTarget extends ListBase {
    public $name = 'Ada';
    private $secret = 's';
    public static $kind = 'user';
    private static $privateKind = 'secret-kind';

    public function run() {}
    private function secretMethod() {}

    public function inside() {
        $vars = get_object_vars($this);
        $classVars = get_class_vars('ListTarget');
        $methods = get_class_methods('ListTarget');
        return $vars['secret'] . ':' . $classVars['secret'] . ':' . (in_array('secretMethod', $methods) ? 'secret' : 'no');
    }
}

$obj = new ListTarget();
$obj->dynamic = 'dyn';

$vars = get_object_vars($obj);
echo $vars['base'];
echo ':';
echo $vars['name'];
echo ':';
echo $vars['dynamic'];
echo ':';
echo array_key_exists('secret', $vars) ? 'secret' : 'no';

$classVars = get_class_vars('listtarget');
echo ':';
echo $classVars['base'];
echo ':';
echo $classVars['name'];
echo ':';
echo $classVars['kind'];
echo ':';
echo array_key_exists('privateKind', $classVars) ? 'private' : 'no';

$methods = get_class_methods($obj);
echo ':';
echo in_array('run', $methods) ? 'run' : 'no';
echo ':';
echo in_array('secretMethod', $methods) ? 'secret' : 'no';
echo ':';
echo $obj->inside();
--EXPECT--
base:Ada:dyn:no:base:Ada:user:no:run:no:s:s:secret
