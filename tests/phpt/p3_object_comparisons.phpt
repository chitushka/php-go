--TEST--
P3 object loose and strict comparisons
--FILE--
<?php
class CompareBase {
    protected $hidden = 'h';
}

class CompareBox extends CompareBase {
    public $name = 'Ada';
    private $secret = 's';

    public function set($name, $secret) {
        $this->name = $name;
        $this->secret = $secret;
    }
}

class CompareOther {
    public $name = 'Ada';
}

$a = new CompareBox();
$b = new CompareBox();
$c = new CompareBox();
$c->set('Ada', 't');
$other = new CompareOther();

echo ($a == $b ? 'eq' : 'bad');
echo ':';
echo ($a === $b ? 'bad' : 'not-ident');
echo ':';
echo ($a === $a ? 'ident' : 'bad');
echo ':';
echo ($a != $c ? 'diff' : 'bad');
echo "\n";
echo ($a == $other ? 'bad' : 'class');
echo ':';
echo ($a < $c ? 'less' : 'bad');
echo ':';
echo ($a > 0 ? 'object-greater' : 'bad');
?>
--EXPECT--
eq:not-ident:ident:diff
class:less:object-greater
